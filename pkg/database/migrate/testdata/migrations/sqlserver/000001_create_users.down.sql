IF EXISTS (SELECT * FROM sysobjects WHERE name='users' AND xtype='U')
DROP TABLE users;
