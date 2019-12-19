package sqlximpl

import sqldb "github.com/domonda/go-sqldb"

// ListenerEventLogger will log all subscribed channel listener events if not nil
var ListenerEventLogger sqldb.Logger
