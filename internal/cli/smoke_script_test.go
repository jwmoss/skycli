package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

func TestSmokeOnlySkipsKnownUnavailableEndpoints(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX smoke script")
	}
	for _, tool := range []string{"bash", "python3"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	stub := filepath.Join(t.TempDir(), "fake-cli.py")
	if err := os.WriteFile(stub, []byte(`import json, os, sys
args = sys.argv[1:]
if "routines" in args:
    status = int(os.environ["FIXTURE_STATUS"])
    if status == 200:
        print("malformed json")
        sys.exit(0)
    print(json.dumps({"error":"fixture failure", "http_status":status}))
    sys.exit(1)
print('[{"id":"123"}]' if "frames" in args and "list" in args else '{}')
`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{403, 404, 401, 429, 500, 200} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			cmd := exec.Command("bash", "../../scripts/live-readonly-smoke.sh")
			cmd.Env = append(os.Environ(), "SKYCLI_BIN=python3 "+stub, "FIXTURE_STATUS="+strconv.Itoa(status), "SKYCLI_FRAME_ID=123")
			out, err := cmd.CombinedOutput()
			wantSuccess := status == 403 || status == 404
			if (err == nil) != wantSuccess {
				t.Fatalf("status=%d success=%v expected=%v\n%s", status, err == nil, wantSuccess, out)
			}
		})
	}
}
