package pqconn

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/pqconn"
)

// testLogger is a thread-safe logger that captures log messages for assertions.
type testLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *testLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf(format, v...))
}

func (l *testLogger) Messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.messages...)
}

func (l *testLogger) ContainsSubstring(sub string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, msg := range l.messages {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}

func openHelperDB(t *testing.T) *sql.DB {
	t.Helper()
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, postgresHost, postgresPort, dbName)
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// getBackendPIDs returns the set of all client backend PIDs
// for the current database.
func getBackendPIDs(t *testing.T, db *sql.DB) map[int]bool {
	t.Helper()
	rows, err := db.Query( /*sql*/ `
		SELECT pid
		FROM pg_stat_activity
		WHERE datname = current_database()
		  AND backend_type = 'client backend'`,
	)
	require.NoError(t, err)
	defer rows.Close()
	pids := make(map[int]bool)
	for rows.Next() {
		var pid int
		require.NoError(t, rows.Scan(&pid))
		pids[pid] = true
	}
	require.NoError(t, rows.Err())
	return pids
}

// findNewBackendPID polls pg_stat_activity until a backend PID
// appears that was not in oldPIDs and returns it.
func findNewBackendPID(t *testing.T, db *sql.DB, oldPIDs map[int]bool) int {
	t.Helper()
	var newPID int
	require.Eventually(t, func() bool {
		for pid := range getBackendPIDs(t, db) {
			if !oldPIDs[pid] {
				newPID = pid
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond)
	return newPID
}

// terminateBackend kills the PostgreSQL backend with the given PID
// using pg_terminate_backend, causing the corresponding connection
// to disconnect. The pq.Listener will then attempt to reconnect.
func terminateBackend(t *testing.T, db *sql.DB, pid int) {
	t.Helper()
	var terminated bool
	err := db.QueryRow( /*sql*/ `SELECT pg_terminate_backend($1)`, pid).Scan(&terminated)
	require.NoError(t, err)
	require.True(t, terminated, "failed to terminate backend %d", pid)
}

func pqConnectWithShortReconnect(t *testing.T) sqldb.Connection {
	t.Helper()
	port, err := strconv.ParseUint(postgresPort, 10, 16)
	require.NoError(t, err)
	config := &sqldb.Config{
		Driver:                       "postgres",
		Host:                         postgresHost,
		Port:                         uint16(port),
		User:                         postgresUser,
		Password:                     postgresPassword,
		Database:                     dbName,
		Extra:                        map[string]string{"sslmode": "disable"},
		ListenerMinReconnectInterval: 100 * time.Millisecond,
		ListenerMaxReconnectInterval: time.Second,
	}
	conn, err := pqconn.Connect(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func pqConnectWithLogger(t *testing.T, errLogger sqldb.Logger) sqldb.Connection {
	t.Helper()
	port, err := strconv.ParseUint(postgresPort, 10, 16)
	require.NoError(t, err)
	config := &sqldb.Config{
		Driver:    "postgres",
		Host:      postgresHost,
		Port:      uint16(port),
		User:      postgresUser,
		Password:  postgresPassword,
		Database:  dbName,
		Extra:     map[string]string{"sslmode": "disable"},
		ErrLogger: errLogger,
	}
	conn, err := pqconn.Connect(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestListenerReconnect(t *testing.T) {
	// given
	conn := pqConnectWithShortReconnect(t)
	listenerConn := conn.(sqldb.ListenerConnection)
	helperDB := openHelperDB(t)

	pidsBefore := getBackendPIDs(t, helperDB)

	notifyCh := make(chan string, 10)
	err := listenerConn.ListenOnChannel("test_reconnect",
		func(_, payload string) { notifyCh <- payload },
		nil,
	)
	require.NoError(t, err)
	assert.True(t, listenerConn.IsListeningOnChannel("test_reconnect"))

	listenerPID := findNewBackendPID(t, helperDB, pidsBefore)

	// Verify notifications work before disconnect
	_, err = helperDB.Exec("NOTIFY test_reconnect, 'before'")
	require.NoError(t, err)
	select {
	case payload := <-notifyCh:
		assert.Equal(t, "before", payload)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for notification before disconnect")
	}

	// when - terminate the listener's backend to force reconnect
	terminateBackend(t, helperDB, listenerPID)

	// then - after reconnect, notifications should be received again
	// Poll by sending NOTIFY until the reconnected listener picks it up.
	require.Eventually(t, func() bool {
		helperDB.Exec("NOTIFY test_reconnect, 'after_reconnect'") //#nosec G104
		time.Sleep(100 * time.Millisecond)
		select {
		case <-notifyCh:
			return true
		default:
			return false
		}
	}, 10*time.Second, 200*time.Millisecond, "notification not received after reconnect")

	// Unlisten should still work after reconnect
	err = listenerConn.UnlistenChannel("test_reconnect")
	require.NoError(t, err)
	assert.False(t, listenerConn.IsListeningOnChannel("test_reconnect"))
}

func TestListenerUnlistenNotResubscribedAfterReconnect(t *testing.T) {
	// given
	conn := pqConnectWithShortReconnect(t)
	listenerConn := conn.(sqldb.ListenerConnection)
	helperDB := openHelperDB(t)

	pidsBefore := getBackendPIDs(t, helperDB)

	chKept := make(chan string, 10)
	chRemoved := make(chan string, 10)

	err := listenerConn.ListenOnChannel("test_kept",
		func(_, payload string) { chKept <- payload },
		nil,
	)
	require.NoError(t, err)

	err = listenerConn.ListenOnChannel("test_removed",
		func(_, payload string) { chRemoved <- payload },
		nil,
	)
	require.NoError(t, err)

	listenerPID := findNewBackendPID(t, helperDB, pidsBefore)

	// Verify both channels receive notifications
	_, err = helperDB.Exec("NOTIFY test_kept, 'k1'")
	require.NoError(t, err)
	_, err = helperDB.Exec("NOTIFY test_removed, 'r1'")
	require.NoError(t, err)
	for range 2 {
		select {
		case <-chKept:
		case <-chRemoved:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for initial notifications")
		}
	}

	// Unlisten one channel before forcing reconnect
	err = listenerConn.UnlistenChannel("test_removed")
	require.NoError(t, err)
	assert.False(t, listenerConn.IsListeningOnChannel("test_removed"))
	assert.True(t, listenerConn.IsListeningOnChannel("test_kept"))

	// when - terminate the listener's backend to force reconnect
	terminateBackend(t, helperDB, listenerPID)

	// then - only the kept channel should be re-subscribed
	require.Eventually(t, func() bool {
		helperDB.Exec("NOTIFY test_kept, 'k2'") //#nosec G104
		time.Sleep(100 * time.Millisecond)
		select {
		case <-chKept:
			return true
		default:
			return false
		}
	}, 10*time.Second, 200*time.Millisecond, "kept channel not re-subscribed after reconnect")

	// The removed channel should NOT receive notifications
	_, err = helperDB.Exec("NOTIFY test_removed, 'r2'")
	require.NoError(t, err)
	time.Sleep(time.Second)
	select {
	case payload := <-chRemoved:
		t.Fatalf("removed channel should not receive notifications after reconnect, got: %s", payload)
	default:
		// expected — channel was not re-subscribed
	}

	// Cleanup
	err = listenerConn.UnlistenChannel("test_kept")
	require.NoError(t, err)
}

func TestListenerMultipleCallbacksSameChannel(t *testing.T) {
	// given
	conn := pqConnect(t)
	listenerConn := conn.(sqldb.ListenerConnection)
	helperDB := openHelperDB(t)

	notifyCh1 := make(chan string, 10)
	notifyCh2 := make(chan string, 10)

	// when - register two callbacks on the same channel
	err := listenerConn.ListenOnChannel("test_multi_cb",
		func(_, payload string) { notifyCh1 <- payload },
		nil,
	)
	require.NoError(t, err)

	err = listenerConn.ListenOnChannel("test_multi_cb",
		func(_, payload string) { notifyCh2 <- payload },
		nil,
	)
	require.NoError(t, err, "second ListenOnChannel on same channel must not error")

	assert.True(t, listenerConn.IsListeningOnChannel("test_multi_cb"))

	// then - both callbacks should receive the notification
	_, err = helperDB.Exec("NOTIFY test_multi_cb, 'hello'")
	require.NoError(t, err)

	select {
	case payload := <-notifyCh1:
		assert.Equal(t, "hello", payload)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for notification on callback 1")
	}
	select {
	case payload := <-notifyCh2:
		assert.Equal(t, "hello", payload)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for notification on callback 2")
	}

	// Cleanup - unlisten removes all callbacks for the channel
	err = listenerConn.UnlistenChannel("test_multi_cb")
	require.NoError(t, err)
	assert.False(t, listenerConn.IsListeningOnChannel("test_multi_cb"))
}

func TestListenerCloseCallsUnlistenCallbacks(t *testing.T) {
	// given - create a dedicated connection so closing it doesn't
	// affect other tests' shared listener
	port, err := strconv.ParseUint(postgresPort, 10, 16)
	require.NoError(t, err)
	config := &sqldb.Config{
		Driver:   "postgres",
		Host:     postgresHost,
		Port:     uint16(port),
		User:     postgresUser,
		Password: postgresPassword,
		Database: dbName,
		Extra:    map[string]string{"sslmode": "disable", "application_name": "test_close_unlisten"},
	}
	conn, err := pqconn.Connect(t.Context(), config)
	require.NoError(t, err)
	listenerConn := conn.(sqldb.ListenerConnection)

	unlistenCh := make(chan string, 10)
	err = listenerConn.ListenOnChannel("test_close_cb",
		nil,
		func(channel string) { unlistenCh <- channel },
	)
	require.NoError(t, err)
	assert.True(t, listenerConn.IsListeningOnChannel("test_close_cb"))

	// when - close the connection
	err = conn.Close()
	require.NoError(t, err)

	// then - unlisten callback should have been called
	select {
	case channel := <-unlistenCh:
		assert.Equal(t, "test_close_cb", channel)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for unlisten callback on close")
	}
}

func TestListenerOnlyUnlistenCallback(t *testing.T) {
	// given - listen with nil onNotify but non-nil onUnlisten
	conn := pqConnect(t)
	listenerConn := conn.(sqldb.ListenerConnection)

	unlistenCh := make(chan string, 10)
	err := listenerConn.ListenOnChannel("test_only_unlisten",
		nil,
		func(channel string) { unlistenCh <- channel },
	)
	require.NoError(t, err)
	assert.True(t, listenerConn.IsListeningOnChannel("test_only_unlisten"))

	// when
	err = listenerConn.UnlistenChannel("test_only_unlisten")
	require.NoError(t, err)

	// then
	select {
	case channel := <-unlistenCh:
		assert.Equal(t, "test_only_unlisten", channel)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for unlisten callback")
	}
	assert.False(t, listenerConn.IsListeningOnChannel("test_only_unlisten"))
}

func TestListenerNotifyCallbackPanic(t *testing.T) {
	// given - two callbacks on the same channel, one panics
	errLog := &testLogger{}
	conn := pqConnectWithLogger(t, errLog)
	listenerConn := conn.(sqldb.ListenerConnection)
	helperDB := openHelperDB(t)

	goodCh := make(chan string, 10)
	err := listenerConn.ListenOnChannel("test_notify_panic",
		func(_, payload string) { panic("boom from notify") },
		nil,
	)
	require.NoError(t, err)

	err = listenerConn.ListenOnChannel("test_notify_panic",
		func(_, payload string) { goodCh <- payload },
		nil,
	)
	require.NoError(t, err)

	// when - send a notification
	_, err = helperDB.Exec("NOTIFY test_notify_panic, 'hello'")
	require.NoError(t, err)

	// then - the non-panicking callback still receives the notification
	select {
	case payload := <-goodCh:
		assert.Equal(t, "hello", payload)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for notification on non-panicking callback")
	}

	// and the panic was logged
	require.Eventually(t, func() bool {
		return errLog.ContainsSubstring("notify callback on channel")
	}, 5*time.Second, 50*time.Millisecond, "expected panic log from notify callback")

	// Cleanup
	err = listenerConn.UnlistenChannel("test_notify_panic")
	require.NoError(t, err)
}

func TestListenerUnlistenCallbackPanic(t *testing.T) {
	// given - a panicking and a normal unlisten callback
	errLog := &testLogger{}
	conn := pqConnectWithLogger(t, errLog)
	listenerConn := conn.(sqldb.ListenerConnection)

	goodCh := make(chan string, 10)
	err := listenerConn.ListenOnChannel("test_unlisten_panic",
		nil,
		func(channel string) { panic("boom from unlisten") },
	)
	require.NoError(t, err)

	err = listenerConn.ListenOnChannel("test_unlisten_panic",
		nil,
		func(channel string) { goodCh <- channel },
	)
	require.NoError(t, err)

	// when
	err = listenerConn.UnlistenChannel("test_unlisten_panic")
	require.NoError(t, err)

	// then - the non-panicking unlisten callback was called
	select {
	case channel := <-goodCh:
		assert.Equal(t, "test_unlisten_panic", channel)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for non-panicking unlisten callback")
	}

	// and the panic was logged
	require.Eventually(t, func() bool {
		return errLog.ContainsSubstring("unlisten callback on channel")
	}, 5*time.Second, 50*time.Millisecond, "expected panic log from unlisten callback")

	assert.False(t, listenerConn.IsListeningOnChannel("test_unlisten_panic"))
}

func TestListenerCloseWithPanickingUnlistenCallback(t *testing.T) {
	// given - a dedicated connection so closing doesn't affect other tests
	errLog := &testLogger{}
	port, err := strconv.ParseUint(postgresPort, 10, 16)
	require.NoError(t, err)
	config := &sqldb.Config{
		Driver:    "postgres",
		Host:      postgresHost,
		Port:      uint16(port),
		User:      postgresUser,
		Password:  postgresPassword,
		Database:  dbName,
		Extra:     map[string]string{"sslmode": "disable"},
		ErrLogger: errLog,
	}
	conn, err := pqconn.Connect(t.Context(), config)
	require.NoError(t, err)
	listenerConn := conn.(sqldb.ListenerConnection)

	goodCh := make(chan string, 10)
	err = listenerConn.ListenOnChannel("test_close_panic",
		nil,
		func(channel string) { panic("boom from close") },
	)
	require.NoError(t, err)

	err = listenerConn.ListenOnChannel("test_close_panic",
		nil,
		func(channel string) { goodCh <- channel },
	)
	require.NoError(t, err)

	// when - close the connection
	err = conn.Close()
	require.NoError(t, err)

	// then - the non-panicking unlisten callback was called
	select {
	case channel := <-goodCh:
		assert.Equal(t, "test_close_panic", channel)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for non-panicking unlisten callback on close")
	}

	// and the panic was logged
	require.Eventually(t, func() bool {
		return errLog.ContainsSubstring("unlisten callback on channel")
	}, 5*time.Second, 50*time.Millisecond, "expected panic log from unlisten callback on close")
}
