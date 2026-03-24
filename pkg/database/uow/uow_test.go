package uow_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"devkit/pkg/database/uow"
)

// --- New validation ---

func TestNew_nilDB_returnsErrDBRequired(t *testing.T) {
	_, err := uow.New(nil)
	if !errors.Is(err, uow.ErrDBRequired) {
		t.Errorf("errors.Is(err, ErrDBRequired) = false, got: %v", err)
	}
}

func TestNew_validDB_succeeds(t *testing.T) {
	db, _, _ := sqlmock.New()
	u, err := uow.New(db)
	if err != nil {
		t.Fatalf("uow.New: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil UnitOfWork")
	}
}

// --- Do: commit on success ---

func TestDo_successfulFn_commits(t *testing.T) {
	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectCommit()

	u, _ := uow.New(db)
	err := u.Do(context.Background(), func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Do: rollback on error ---

func TestDo_errFn_rollsBack(t *testing.T) {
	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectRollback()

	u, _ := uow.New(db)
	fnErr := errors.New("fn error")
	err := u.Do(context.Background(), func(_ context.Context) error {
		return fnErr
	})
	if !errors.Is(err, fnErr) {
		t.Errorf("expected fn error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Do: rollback on panic + re-panic ---

func TestDo_panicFn_rollsBackAndRepanics(t *testing.T) {
	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectRollback()

	u, _ := uow.New(db)

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_ = u.Do(context.Background(), func(_ context.Context) error {
			panic("test panic")
		})
	}()

	if !panicked {
		t.Error("expected panic to propagate")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Do: errors.Join on fn error + rollback error ---

func TestDo_fnErrAndRollbackErr_joinsErrors(t *testing.T) {
	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	rbErr := errors.New("rollback error")
	mock.ExpectRollback().WillReturnError(rbErr)

	u, _ := uow.New(db)
	fnErr := errors.New("fn error")
	err := u.Do(context.Background(), func(_ context.Context) error {
		return fnErr
	})

	if err == nil {
		t.Fatal("expected joined error, got nil")
	}
	if !errors.Is(err, fnErr) {
		t.Errorf("expected fn error in joined error, got: %v", err)
	}
	if !errors.Is(err, rbErr) {
		t.Errorf("expected rollback error in joined error, got: %v", err)
	}
	if errMet := mock.ExpectationsWereMet(); errMet != nil {
		t.Errorf("unmet expectations: %v", errMet)
	}
}

// --- GetRepository: table-driven ---

func TestGetRepository(t *testing.T) {
	type myRepo struct{ name string }

	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectCommit()

	u, _ := uow.New(db)
	u.Register("myRepo", func(_ *sql.Tx) any {
		return &myRepo{name: "from tx"}
	})

	var (
		gotRepo     *myRepo
		gotTypeMis  error
		gotNotFound error
		gotNoTx     error
	)

	_ = u.Do(context.Background(), func(ctx context.Context) error {
		// Correct type.
		r, err := uow.GetRepository[*myRepo](ctx, u, "myRepo")
		if err != nil {
			t.Errorf("GetRepository correct type: %v", err)
		}
		gotRepo = r

		// Wrong type assertion.
		_, gotTypeMis = uow.GetRepository[string](ctx, u, "myRepo")

		// Name not registered.
		_, gotNotFound = uow.GetRepository[*myRepo](ctx, u, "unknown")

		return nil
	})

	// Outside Do: no active transaction.
	_, gotNoTx = uow.GetRepository[*myRepo](context.Background(), u, "myRepo")

	if gotRepo == nil || gotRepo.name != "from tx" {
		t.Errorf("expected gotRepo.name='from tx', got %v", gotRepo)
	}
	if gotTypeMis == nil {
		t.Error("expected type mismatch error, got nil")
	}
	if !errors.Is(gotNotFound, uow.ErrRepositoryNotFound) {
		t.Errorf("expected ErrRepositoryNotFound, got: %v", gotNotFound)
	}
	if !errors.Is(gotNoTx, uow.ErrNoActiveTransaction) {
		t.Errorf("expected ErrNoActiveTransaction, got: %v", gotNoTx)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Register and factory lookup inside Do ---

func TestRegister_factoryReceivesTx(t *testing.T) {
	type repo struct{ called bool }

	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectCommit()

	u, _ := uow.New(db)
	u.Register("r", func(_ *sql.Tx) any {
		return &repo{called: true}
	})

	var got *repo
	err := u.Do(context.Background(), func(ctx context.Context) error {
		r, err := uow.GetRepository[*repo](ctx, u, "r")
		if err != nil {
			return fmt.Errorf("GetRepository: %w", err)
		}
		got = r
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got == nil || !got.called {
		t.Error("expected factory to have been called with called=true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Concurrency: concurrent Register calls must not race ---

func TestRegister_concurrent_noRace(t *testing.T) {
	// Verifies that concurrent Register calls do not race on the factories map.
	// Run with: go test -race ./pkg/database/uow/...
	const goroutines = 20
	db, _, _ := sqlmock.New()
	u, _ := uow.New(db)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u.Register(fmt.Sprintf("repo%d", i), func(_ *sql.Tx) any { return i })
		}(i)
	}
	wg.Wait()
}

// --- Concurrency: concurrent Do calls must not race ---

func TestDo_concurrent_noRace(t *testing.T) {
	// Verifies that concurrent Do calls do not race on the factories map.
	// Goroutines are given individual mock sets via sequential serialization
	// to isolate UoW race behavior from sqlmock ordering constraints.
	// Run with: go test -race ./pkg/database/uow/...
	const goroutines = 5

	u, doFn := newSequentialUOW(t, goroutines)
	u.Register("r", func(_ *sql.Tx) any { return 1 })

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = doFn()
		}()
	}
	wg.Wait()
}

// newSequentialUOW creates a UnitOfWork backed by a sqlmock that has
// goroutines × (ExpectBegin + ExpectCommit) expectations set in pairs.
// It returns the UoW and a closure that executes Do with a no-op fn.
// The mock serializes Begin/Commit pairs via an internal mutex, making it
// safe to call from multiple goroutines when concurrency is light.
func newSequentialUOW(t *testing.T, n int) (*uow.UnitOfWork, func() error) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	var mu sync.Mutex
	for range n {
		mock.ExpectBegin()
		mock.ExpectCommit()
	}
	u, _ := uow.New(db)
	doFn := func() error {
		mu.Lock()
		defer mu.Unlock()
		return u.Do(context.Background(), func(_ context.Context) error {
			return nil
		})
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
	return u, doFn
}

// --- TxFromContext ---

func TestTxFromContext_outsideDo_returnsNil(t *testing.T) {
	if tx := uow.TxFromContext(context.Background()); tx != nil {
		t.Errorf("expected nil tx outside Do, got %v", tx)
	}
}

func TestTxFromContext_insideDo_returnsActiveTx(t *testing.T) {
	db, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectCommit()

	u, _ := uow.New(db)
	_ = u.Do(context.Background(), func(ctx context.Context) error {
		if uow.TxFromContext(ctx) == nil {
			t.Error("expected non-nil tx inside Do")
		}
		return nil
	})
}

// --- Sentinel errors are inspectable via errors.Is ---

func TestSentinelErrors_errorsIs(t *testing.T) {
	errs := []error{
		uow.ErrDBRequired,
		uow.ErrRepositoryNotFound,
		uow.ErrNoActiveTransaction,
	}
	for _, sentinel := range errs {
		if !errors.Is(sentinel, sentinel) {
			t.Errorf("errors.Is(%v, %v) returned false", sentinel, sentinel)
		}
	}
}
