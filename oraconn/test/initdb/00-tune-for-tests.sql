-- Disable recyclebin for faster DROP TABLE operations
ALTER SYSTEM SET RECYCLEBIN=OFF SCOPE=BOTH;

-- Raise PROCESSES to comfortably accommodate the full conntest suite,
-- which opens a fresh connection per sub-test. The previous value of 200
-- caused intermittent ORA-12516 ("listener could not find available
-- handler") when many sub-tests ran in sequence and Oracle's internal
-- background processes plus user sessions exceeded the limit.
-- SESSIONS auto-derives from PROCESSES (Oracle uses 1.5 * PROCESSES + 22).
ALTER SYSTEM SET PROCESSES=400 SCOPE=SPFILE;
