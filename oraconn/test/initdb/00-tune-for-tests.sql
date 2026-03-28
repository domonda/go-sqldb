-- Disable recyclebin for faster DROP TABLE operations
ALTER SYSTEM SET RECYCLEBIN=OFF SCOPE=BOTH;

-- Reduce processes limit (default is too high for tests)
ALTER SYSTEM SET PROCESSES=200 SCOPE=SPFILE;
