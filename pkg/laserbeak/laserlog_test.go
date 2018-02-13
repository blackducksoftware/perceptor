package laserbeak

import "testing"

func TestLogging(t *testing.T) {
	ZLogInfo(LZ_CATCH_ALL, "This is a log message")
}
