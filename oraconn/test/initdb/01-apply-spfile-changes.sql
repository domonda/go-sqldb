-- Restart the instance so SPFILE changes from 00-tune-for-tests.sql
-- (notably PROCESSES) take effect on first container start. Without this
-- the in-memory parameter values stay at the image defaults and tests
-- can hit ORA-12516 under burst connection churn.
SHUTDOWN IMMEDIATE;
STARTUP;
