package hosts

import (
	"strings"
	"testing"
)

func TestValidateIPValid(t *testing.T) {
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "::1", "fe80::1"} {
		if err := ValidateIP(ip); err != nil {
			t.Errorf("expected valid IP %q, got error: %v", ip, err)
		}
	}
}

func TestValidateIPInvalid(t *testing.T) {
	for _, ip := range []string{"", "not-an-ip", "256.1.1.1", "192.168.1"} {
		if err := ValidateIP(ip); err == nil {
			t.Errorf("expected error for invalid IP %q", ip)
		}
	}
}

func TestValidateHostnameValid(t *testing.T) {
	for _, h := range []string{"devbox", "devbox.local", "my-vm", "a", "test.example.com"} {
		if err := ValidateHostname(h); err != nil {
			t.Errorf("expected valid hostname %q, got error: %v", h, err)
		}
	}
}

func TestValidateHostnameInvalid(t *testing.T) {
	for _, h := range []string{"", "-start", "end-", ".dot", "has space", "has@symbol"} {
		if err := ValidateHostname(h); err == nil {
			t.Errorf("expected error for invalid hostname %q", h)
		}
	}
}

func TestValidateHostnameTooLong(t *testing.T) {
	long := strings.Repeat("a", 254)
	if err := ValidateHostname(long); err == nil {
		t.Error("expected error for hostname exceeding 253 characters")
	}
}

func TestBuildHostnamesWithDNSHostnameAndDomain(t *testing.T) {
	result := BuildHostnames("myvm", "devbox", "example.com")
	expected := []string{"devbox.example.com", "devbox"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d hostnames, got %d: %v", len(expected), len(result), result)
	}
	for i, h := range expected {
		if result[i] != h {
			t.Errorf("hostname[%d]: expected %q, got %q", i, h, result[i])
		}
	}
}

func TestBuildHostnamesWithDNSHostnameNoDomain(t *testing.T) {
	result := BuildHostnames("myvm", "devbox", "")
	if len(result) != 1 || result[0] != "devbox" {
		t.Errorf("expected [devbox], got %v", result)
	}
}

func TestBuildHostnamesFallbackToVMName(t *testing.T) {
	result := BuildHostnames("myvm", "", "example.com")
	expected := []string{"myvm.example.com", "myvm"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d hostnames, got %d: %v", len(expected), len(result), result)
	}
	for i, h := range expected {
		if result[i] != h {
			t.Errorf("hostname[%d]: expected %q, got %q", i, h, result[i])
		}
	}
}

func TestBuildHostnamesVMNameOnly(t *testing.T) {
	result := BuildHostnames("myvm", "", "")
	if len(result) != 1 || result[0] != "myvm" {
		t.Errorf("expected [myvm], got %v", result)
	}
}

func TestBuildBlock(t *testing.T) {
	block := buildBlock("devbox", "192.168.64.13", []string{"devbox.local", "devbox"})
	expected := "# goloo:devbox\n192.168.64.13    devbox.local devbox\n# /goloo:devbox\n"
	if block != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, block)
	}
}

func TestBuildBlockSingleHostname(t *testing.T) {
	block := buildBlock("myvm", "10.0.0.1", []string{"myvm"})
	expected := "# goloo:myvm\n10.0.0.1    myvm\n# /goloo:myvm\n"
	if block != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, block)
	}
}

func TestRemoveBlockPresent(t *testing.T) {
	content := "127.0.0.1 localhost\n# goloo:devbox\n192.168.64.13    devbox.local devbox\n# /goloo:devbox\n::1 localhost\n"
	result := removeBlock(content, "devbox")
	expected := "127.0.0.1 localhost\n::1 localhost\n"
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestRemoveBlockNotPresent(t *testing.T) {
	content := "127.0.0.1 localhost\n::1 localhost\n"
	result := removeBlock(content, "devbox")
	if result != content {
		t.Errorf("expected content unchanged, got:\n%q", result)
	}
}

func TestRemoveBlockMultipleVMs(t *testing.T) {
	content := "127.0.0.1 localhost\n" +
		"# goloo:vm1\n10.0.0.1    vm1\n# /goloo:vm1\n" +
		"# goloo:vm2\n10.0.0.2    vm2\n# /goloo:vm2\n"
	result := removeBlock(content, "vm1")
	if strings.Contains(result, "vm1") {
		t.Error("expected vm1 block removed")
	}
	if !strings.Contains(result, "goloo:vm2") {
		t.Error("expected vm2 block preserved")
	}
}

func TestRemoveBlockOnlyTargeted(t *testing.T) {
	content := "# goloo:keep\n10.0.0.1    keep\n# /goloo:keep\n" +
		"# goloo:remove\n10.0.0.2    remove\n# /goloo:remove\n"
	result := removeBlock(content, "remove")
	if !strings.Contains(result, "goloo:keep") {
		t.Error("expected 'keep' block preserved")
	}
	if strings.Contains(result, "goloo:remove") {
		t.Error("expected 'remove' block removed")
	}
}
