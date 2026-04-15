package fake

import (
	"context"
	"sync"
	"time"

	observability "devkit/pkg/o11y"
)

// FakeMetrics stores metric instruments and observations in memory.
type FakeMetrics struct {
	mu         sync.RWMutex
	counters   map[string]*FakeCounter
	histograms map[string]*FakeHistogram
	upDowns    map[string]*FakeUpDownCounter
	gauges     map[string]*FakeGauge
}

// NewFakeMetrics creates an in-memory metrics registry for tests.
func NewFakeMetrics() *FakeMetrics {
	return &FakeMetrics{
		counters:   make(map[string]*FakeCounter),
		histograms: make(map[string]*FakeHistogram),
		upDowns:    make(map[string]*FakeUpDownCounter),
		gauges:     make(map[string]*FakeGauge),
	}
}

// Counter returns an existing or new in-memory counter for the given name.
func (m *FakeMetrics) Counter(name, description, unit string) (observability.Counter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, exists := m.counters[name]; exists {
		return c, nil
	}

	c := &FakeCounter{
		Name:        name,
		Description: description,
		Unit:        unit,
		values:      make([]CounterValue, 0),
	}
	m.counters[name] = c
	return c, nil
}

// Histogram returns an existing or new in-memory histogram for the given name.
func (m *FakeMetrics) Histogram(name, description, unit string) (observability.Histogram, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, exists := m.histograms[name]; exists {
		return h, nil
	}

	h := &FakeHistogram{
		Name:        name,
		Description: description,
		Unit:        unit,
		values:      make([]HistogramValue, 0),
	}
	m.histograms[name] = h
	return h, nil
}

// UpDownCounter returns an existing or new in-memory up-down counter.
func (m *FakeMetrics) UpDownCounter(name, description, unit string) (observability.UpDownCounter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u, exists := m.upDowns[name]; exists {
		return u, nil
	}

	u := &FakeUpDownCounter{
		Name:        name,
		Description: description,
		Unit:        unit,
		values:      make([]CounterValue, 0),
	}
	m.upDowns[name] = u
	return u, nil
}

// Gauge registers an asynchronous gauge backed by the given callback.
func (m *FakeMetrics) Gauge(name, description, unit string, callback observability.GaugeCallback) error {
	if callback == nil {
		return observability.ErrNilGaugeCallback
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[name] = &FakeGauge{
		Name:        name,
		Description: description,
		Unit:        unit,
		callback:    callback,
	}

	return nil
}

// GetCounter returns the registered fake counter by name.
func (m *FakeMetrics) GetCounter(name string) *FakeCounter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

// GetHistogram returns the registered fake histogram by name.
func (m *FakeMetrics) GetHistogram(name string) *FakeHistogram {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.histograms[name]
}

// GetUpDownCounter returns the registered fake up-down counter by name.
func (m *FakeMetrics) GetUpDownCounter(name string) *FakeUpDownCounter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.upDowns[name]
}

// GetGauge returns the registered fake gauge by name.
func (m *FakeMetrics) GetGauge(name string) *FakeGauge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

// FakeCounter records received counter increments.
type FakeCounter struct {
	mu          sync.RWMutex
	Name        string
	Description string
	Unit        string
	values      []CounterValue
}

// Add records a counter increment with the given value.
func (c *FakeCounter) Add(ctx context.Context, value int64, fields ...observability.Field) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = append(c.values, CounterValue{
		Value:     value,
		Fields:    cloneFakeFields(fields),
		Timestamp: time.Now(),
	})
}

// Increment records a counter increment of 1.
func (c *FakeCounter) Increment(ctx context.Context, fields ...observability.Field) {
	c.Add(ctx, 1, fields...)
}

// GetValues returns a copy of the recorded counter values.
func (c *FakeCounter) GetValues() []CounterValue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]CounterValue, len(c.values))
	for i, value := range c.values {
		result[i] = CounterValue{
			Value:     value.Value,
			Fields:    cloneFakeFields(value.Fields),
			Timestamp: value.Timestamp,
		}
	}
	return result
}

// CounterValue represents a recorded counter value.
type CounterValue struct {
	Value     int64
	Fields    []observability.Field
	Timestamp time.Time
}

// FakeHistogram records observed histogram samples.
type FakeHistogram struct {
	mu          sync.RWMutex
	Name        string
	Description string
	Unit        string
	values      []HistogramValue
}

// Record records a histogram observation.
func (h *FakeHistogram) Record(ctx context.Context, value float64, fields ...observability.Field) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, HistogramValue{
		Value:     value,
		Fields:    cloneFakeFields(fields),
		Timestamp: time.Now(),
	})
}

// GetValues returns a copy of the recorded histogram values.
func (h *FakeHistogram) GetValues() []HistogramValue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]HistogramValue, len(h.values))
	for i, value := range h.values {
		result[i] = HistogramValue{
			Value:     value.Value,
			Fields:    cloneFakeFields(value.Fields),
			Timestamp: value.Timestamp,
		}
	}
	return result
}

// HistogramValue represents a recorded histogram sample.
type HistogramValue struct {
	Value     float64
	Fields    []observability.Field
	Timestamp time.Time
}

// FakeUpDownCounter records positive and negative deltas.
type FakeUpDownCounter struct {
	mu          sync.RWMutex
	Name        string
	Description string
	Unit        string
	values      []CounterValue
}

// Add records a delta (positive or negative) on the up-down counter.
func (u *FakeUpDownCounter) Add(ctx context.Context, value int64, fields ...observability.Field) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.values = append(u.values, CounterValue{
		Value:     value,
		Fields:    cloneFakeFields(fields),
		Timestamp: time.Now(),
	})
}

// GetValues returns a copy of the recorded up-down counter values.
func (u *FakeUpDownCounter) GetValues() []CounterValue {
	u.mu.RLock()
	defer u.mu.RUnlock()
	result := make([]CounterValue, len(u.values))
	for i, value := range u.values {
		result[i] = CounterValue{
			Value:     value.Value,
			Fields:    cloneFakeFields(value.Fields),
			Timestamp: value.Timestamp,
		}
	}
	return result
}

// FakeGauge stores a registered asynchronous gauge callback for tests.
type FakeGauge struct {
	Name        string
	Description string
	Unit        string
	callback    observability.GaugeCallback
}

// Observe evaluates the registered callback with the provided context.
func (g *FakeGauge) Observe(ctx context.Context) float64 {
	if ctx == nil {
		ctx = context.Background()
	}

	return g.callback(ctx)
}
