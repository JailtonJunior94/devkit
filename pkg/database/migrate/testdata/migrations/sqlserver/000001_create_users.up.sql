IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='users' AND xtype='U')
CREATE TABLE users (
    id   INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(255) NOT NULL
);
