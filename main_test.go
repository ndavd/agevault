package main

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testVaultName  = "vault"
	testFooName    = "foo.txt"
	testFooContent = "test"
	testPassphrase = "agevault-test-passphrase"
)

func constPassphrase(string) (string, error) {
	return testPassphrase, nil
}

func runInIsolatedDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatal(err)
		}
	})
}

func assertVaultUnlocked(t *testing.T) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(testVaultName, testFooName))
	if err != nil {
		t.Fatalf("reading unlocked %s: %v", testFooName, err)
	}
	if string(got) != testFooContent {
		t.Fatalf("%s content = %q, want %q", testFooName, got, testFooContent)
	}
}

func TestUnlockFixtures(t *testing.T) {
	fixtures := map[string]string{
		"v1.0.0-zip": "testdata/v1.0.0-zip-vault",
		"current":    "testdata/current-vault",
	}
	for name, dir := range fixtures {
		t.Run(name, func(t *testing.T) {
			fixturePath, err := filepath.Abs(dir)
			if err != nil {
				t.Fatal(err)
			}

			runInIsolatedDir(t)

			entries, err := os.ReadDir(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				data, err := os.ReadFile(filepath.Join(fixturePath, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(entry.Name(), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if err := Unlock(testVaultName, testVaultName, constPassphrase); err != nil {
				t.Fatalf("Unlock() failed: %v", err)
			}

			assertVaultUnlocked(t)
		})
	}
}

func TestLockUnlock(t *testing.T) {
	runInIsolatedDir(t)

	if err := os.Mkdir(testVaultName, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testVaultName, testFooName), []byte(testFooContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Keygen(testVaultName, testPassphrase); err != nil {
		t.Fatalf("Keygen() failed: %v", err)
	}
	if _, err := Lock(testVaultName, testVaultName); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}
	if err := Unlock(testVaultName, testVaultName, constPassphrase); err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	assertVaultUnlocked(t)
}

func TestLockRefusesHardLinks(t *testing.T) {
	runInIsolatedDir(t)

	if err := os.Mkdir(testVaultName, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("outside.txt", []byte(testFooContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link("outside.txt", filepath.Join(testVaultName, testFooName)); err != nil {
		t.Fatal(err)
	}
	if _, err := Keygen(testVaultName, testPassphrase); err != nil {
		t.Fatal(err)
	}

	if _, err := Lock(testVaultName, testVaultName); err == nil {
		t.Fatal("Lock() succeeded, want hard link error")
	}
	if got, _ := os.ReadFile("outside.txt"); string(got) != testFooContent {
		t.Fatalf("outside.txt = %q, want %q", got, testFooContent)
	}
}
