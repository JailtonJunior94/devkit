IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='items' AND xtype='U')
CREATE TABLE items (
    id   INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(255) NOT NULL
);
