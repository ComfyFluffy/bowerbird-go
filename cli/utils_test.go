package cli

import (
	"net/url"
	"testing"
)

const u = "https://i.pximg.net/img-original/img/2012/05/22/16/16/22/27427531_p0.png"

var up, _ = url.Parse(u)

func TestRegexp(t *testing.T) {
	r := pximgDate.FindString(u)
	if r != "2012/05/22/16/16/22" {
		t.Error(r)
	}
}

func TestPximgSingleFileWithDate(t *testing.T) {
	r := pximgSingleFileWithDate("C:\\test", 123, up)
	if r != `C:\test\123\27427531_p0_20120522161622.png` {
		t.Error(r)
	}
}
