package selinuxwhy

import "testing"

func TestParseAVCDenial(t *testing.T) {
	d := parse(`type=AVC msg=audit(1): avc:  denied  { read write } for  pid=42 comm="httpd" name="secret" scontext=a tcontext=b tclass=file`)
	if d.PID != "42" || d.Comm != "httpd" || d.TClass != "file" || len(d.Permissions) != 2 {
		t.Fatalf("denial=%#v", d)
	}
}
