package auth

import (
	"testing"
)

func TestExtractBetween(t *testing.T) {
	str := "{sha256}(foo{password}{user}{salt}{globalsalt})"
	got := extractBetween(str, "{sha256}(", ")")
	want := "foo{password}{user}{salt}{globalsalt}"
	if got != want {
		t.Errorf("extractBetween failed: got %q, want %q", got, want)
	}

	// Test missing start
	got = extractBetween(str, "{sha1}(", ")")
	if got != "" {
		t.Errorf("extractBetween should return empty string if start not found")
	}

	// Test missing end
	got = extractBetween("{sha256}(foo", "{sha256}(", ")")
	if got != "" {
		t.Errorf("extractBetween should return empty string if end not found")
	}
}

func TestSha256Hash(t *testing.T) {
	s := "hello"
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if sha256Hash(s) != expected {
		t.Errorf("sha256Hash failed: got %q, want %q", sha256Hash(s), expected)
	}
}

func TestSha1Hash(t *testing.T) {
	s := "hello"
	expected := "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
	if sha1Hash(s) != expected {
		t.Errorf("sha1Hash failed: got %q, want %q", sha1Hash(s), expected)
	}
}

func TestMd5Hash(t *testing.T) {
	s := "hello"
	expected := "5d41402abc4b2a76b9719d911017c592"
	if md5Hash(s) != expected {
		t.Errorf("md5Hash failed: got %q, want %q", md5Hash(s), expected)
	}
}

func TestApplyHashMacro(t *testing.T) {
	password := "pass"
	user := "bob"
	userSalt := "usalt"
	globalSalt := "gsalt"

	// SHA256
	hash, err := ApplyHashMacro("{sha256}({password}{user}{salt}{globalsalt})", password, user, userSalt, globalSalt)
	if err != nil {
		t.Fatalf("ApplyHashMacro sha256 failed: %v", err)
	}
	expected := sha256Hash(password + user + userSalt + globalSalt)
	if hash != expected {
		t.Errorf("ApplyHashMacro sha256: got %q, want %q", hash, expected)
	}

	// SHA1
	hash, err = ApplyHashMacro("{sha1}({password}{user})", password, user, userSalt, globalSalt)
	if err != nil {
		t.Fatalf("ApplyHashMacro sha1 failed: %v", err)
	}
	expected = sha1Hash(password + user)
	if hash != expected {
		t.Errorf("ApplyHashMacro sha1: got %q, want %q", hash, expected)
	}

	// MD5
	hash, err = ApplyHashMacro("{md5}({user}{salt})", password, user, userSalt, globalSalt)
	if err != nil {
		t.Fatalf("ApplyHashMacro md5 failed: %v", err)
	}
	expected = md5Hash(user + userSalt)
	if hash != expected {
		t.Errorf("ApplyHashMacro md5: got %q, want %q", hash, expected)
	}

	// Clear
	clear, err := ApplyHashMacro("{clear}({password})", password, user, userSalt, globalSalt)
	if err != nil {
		t.Fatalf("ApplyHashMacro clear failed: %v", err)
	}
	if clear != password {
		t.Errorf("ApplyHashMacro clear: got %q, want %q", clear, password)
	}

	// Unsupported
	_, err = ApplyHashMacro("{unknown}({password})", password, user, userSalt, globalSalt)
	if err == nil {
		t.Error("ApplyHashMacro should fail for unsupported macro")
	}
}
