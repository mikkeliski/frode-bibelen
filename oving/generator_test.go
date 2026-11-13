package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestFasitMotGCC kompilerer genererte program med gcc, kjører dem og sjekker
// at utskriften er nøyaktig lik fasiten Go har regnet ut.
// Hoppes over hvis gcc mangler eller med -short. Antall frø per mal kan
// settes med OVING_FRO (standard 60).
func TestFasitMotGCC(t *testing.T) {
	if testing.Short() {
		t.Skip("hopper over gcc-test med -short")
	}
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc finnes ikke")
	}
	antall := 60
	if s := os.Getenv("OVING_FRO"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			antall = n
		}
	}
	for malNr := range maler {
		t.Run(fmt.Sprintf("mal%d", malNr), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for fro := uint64(1); fro <= uint64(antall); fro++ {
				id := LagID(malNr, fro*0x9e3779b97f4a7c15)
				o, err := HentOppgave(id)
				if err != nil {
					t.Fatalf("%s: %v", id, err)
				}
				src := filepath.Join(dir, "p.c")
				bin := filepath.Join(dir, "p")
				if err := os.WriteFile(src, []byte(o.Kode), 0o600); err != nil {
					t.Fatal(err)
				}
				var stderr bytes.Buffer
				cmd := exec.Command(gcc, "-std=c11", "-Wall", "-Wextra", "-Werror", "-o", bin, src)
				cmd.Stderr = &stderr
				if err := cmd.Run(); err != nil {
					t.Fatalf("%s kompilerer ikke: %v\n%s\n%s", id, err, stderr.String(), o.Kode)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				ut, err := exec.CommandContext(ctx, bin).Output()
				cancel()
				if err != nil {
					t.Fatalf("%s: kjøring feilet: %v", id, err)
				}
				if string(ut) != o.Fasit {
					t.Fatalf("%s: feil fasit\n--- kode ---\n%s\n--- gcc ---\n%q\n--- fasit ---\n%q", id, o.Kode, ut, o.Fasit)
				}
			}
		})
	}
}

func TestSammeIDGirSammeOppgave(t *testing.T) {
	for i := 0; i < 50; i++ {
		id := NyID("")
		a, err := HentOppgave(id)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := HentOppgave(id)
		if a.Kode != b.Kode || a.Fasit != b.Fasit || a.ID != id {
			t.Fatalf("%s gir ulike oppgaver", id)
		}
	}
}

func TestNyIDRespekterDel(t *testing.T) {
	for _, del := range []string{"A", "B"} {
		for i := 0; i < 50; i++ {
			o, err := HentOppgave(NyID(del))
			if err != nil || o.Del != del {
				t.Fatalf("NyID(%q) ga del %q (%v)", del, o.Del, err)
			}
		}
	}
}

func TestUgyldigeIDer(t *testing.T) {
	for _, id := range []string{
		"", "x", "1", "1-", "-0000000000000001", "1-123", "1-00000000000000001",
		"99-0000000000000001", "10-0000000000000001", "1-000000000000000G",
		"1-ABCDEF0123456789", "01-0000000000000001x", "1 -0000000000000001",
		"../../etc/passwd", "1-0000000000000001\n",
	} {
		if _, err := HentOppgave(id); err == nil {
			t.Errorf("HentOppgave(%q) skulle feilet", id)
		}
	}
}

// Alle Frodes eksamener med «Hva blir utskriften?» oppgir at det blir 5 linjer.
func TestAlltidFemLinjer(t *testing.T) {
	for malNr := range maler {
		for fro := uint64(1); fro <= 500; fro++ {
			o, err := HentOppgave(LagID(malNr, fro*0x9e3779b97f4a7c15))
			if err != nil {
				t.Fatal(err)
			}
			if o.Linjer() != 5 || !strings.HasSuffix(o.Fasit, "\n") {
				t.Fatalf("%s: %d linjer, ville ha 5\n%q", o.ID, o.Linjer(), o.Fasit)
			}
		}
	}
}
