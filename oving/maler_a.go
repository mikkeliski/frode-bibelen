package main

// Del A: tekst, char-arrayer, pekere og string.h (som oppgave 1a på eksamen).
//
// Hver mal lager C-koden OG regner ut fasiten i Go. generator_test.go
// kompilerer koden med gcc og sjekker at fasiten stemmer.

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

func malIndekshopp(r *rand.Rand) (string, string, []string) {
	txt := lagTekst(r)
	n := len(txt)
	start := mellom(r, 0, 2)
	steg := mellom(r, 2, 5)
	steg2 := mellom(r, 3, 6)
	m := mellom(r, 1, n-2)
	m2 := mellom(r, 0, n-1)

	var ut strings.Builder
	for i := start; i < n; i += steg {
		ut.WriteByte(txt[i])
	}
	ut.WriteString("\n")
	for i := n - 1; i >= 0; i -= steg2 {
		ut.WriteByte(txt[i])
	}
	ut.WriteString("\n")
	fmt.Fprintf(&ut, "%s\n", txt[m:])
	fmt.Fprintf(&ut, "%d\n", len(txt[m2:]))

	kode := fmt.Sprintf(`#include <stdio.h>
#include <string.h>

char txt[] = "%s";

int main()  {
  int i,  len = strlen(txt);

  for (i = %d;  i < len;  i += %d)
      printf("%%c", txt[i]);
  printf("\n");

  for (i = len - 1;  i >= 0;  i -= %d)
      printf("%%c", txt[i]);
  printf("\n");

  printf("%%s\n", txt + %d);
  printf("%%i\n", (int) strlen(txt + %d));
  return 0;
}
`, txt, start, steg, steg2, m, m2)

	return kode, ut.String(), []string{
		"Skriv indeksene under teksten (0, 1, 2, …) og list opp alle verdiene i tar i hver løkke.",
		"txt + m er teksten fra indeks m og ut – printf(\"%s\", txt + m) skriver resten av teksten.",
		"strlen teller ikke med '\\0'.",
	}
}

func malPekere(r *rand.Rand) (string, string, []string) {
	txt := lagTekst(r)
	n := len(txt)
	bok := bokstavIndekser(txt)

	c1 := txt[velg(r, bok)]
	tell := strings.Count(txt, string(c1))
	k := mellom(r, 1, min(6, n))

	p := mellom(r, 0, n-3)
	var c2 byte
	if r.IntN(3) > 0 {
		c2 = txt[mellom(r, p, n-1)] // finnes fra p og ut
		if !erBokstav(c2) {
			c2 = ikkeIBruk(r, txt[p:])
		}
	} else {
		c2 = ikkeIBruk(r, txt[p:])
	}
	steg := strings.IndexByte(txt[p:], c2)
	if steg < 0 {
		steg = n - p
	}
	rest := txt[p+steg:]

	a, b, c := mellom(r, 0, n-1), mellom(r, 0, n-1), mellom(r, 0, n-1)

	var ut strings.Builder
	fmt.Fprintf(&ut, "%d %c\n", tell, txt[n-k])
	fmt.Fprintf(&ut, "%d %s\n", steg, rest)
	fmt.Fprintf(&ut, "%c%c%c\n", txt[a], txt[b], txt[c]+1)

	kode := fmt.Sprintf(`#include <stdio.h>

char txt[] = "%s";

int main()  {
  char* t = txt;
  int   n = 0,  m = 0;

  while (*t)  {
    if (*t++ == %s)  n++;
  }
  printf("%%i %%c\n", n, *(t - %d));

  t = txt + %d;
  while (*t  &&  *t != %s)  {  m++;  t++;  }
  printf("%%i %%s\n", m, t);

  printf("%%c%%c%%c\n", *(txt + %d), txt[%d], *(txt + %d) + 1);
  return 0;
}
`, txt, tegnLiteral(c1), k, p, tegnLiteral(c2), a, b, c)

	return kode, ut.String(), []string{
		"*t++ == 'x' sammenligner tegnet der t står, og flytter t etterpå – også når sammenligningen er usann.",
		"Etter while (*t) står t på '\\0', altså indeks strlen(txt). *(t - k) er da k plasser fra slutten.",
		"Finnes ikke tegnet, stopper den andre løkka på '\\0', og %s skriver en tom tekst.",
		"*(txt + c) + 1 er tegnet rett etter i ASCII-tabellen (vedlegg A).",
	}
}

func malStrchr(r *rand.Rand) (string, string, []string) {
	var txt string
	var c byte
	var forste int
	for {
		txt = lagTekst(r)
		var kand []byte
		for i := 1; i < len(txt)-1; i++ {
			ch := txt[i]
			if (erBokstav(ch) || ch == '-') && strings.IndexByte(txt, ch) == i {
				kand = append(kand, ch)
			}
		}
		if len(kand) > 0 {
			c = velg(r, kand)
			forste = strings.IndexByte(txt, c)
			break
		}
	}
	bok := bokstavIndekser(txt)
	d := txt[velg(r, bok)]
	siste := strings.LastIndexByte(txt, d)

	var ut strings.Builder
	fmt.Fprintf(&ut, "%d %c%c\n", forste, txt[forste-1], txt[forste+1])
	if neste := strings.IndexByte(txt[forste+1:], c); neste >= 0 {
		fmt.Fprintf(&ut, "%s\n", txt[forste+1+neste:])
	} else {
		ut.WriteString("Ingen flere\n")
	}
	fmt.Fprintf(&ut, "%d %s\n", siste, txt[siste+1:])

	kode := fmt.Sprintf(`#include <stdio.h>
#include <string.h>

char txt[] = "%s";

int main()  {
  char* t = strchr(txt, %s);

  printf("%%i %%c%%c\n", (int) (t - txt), *(t - 1), *(t + 1));

  t = strchr(t + 1, %s);
  if (t)  printf("%%s\n", t);
  else    printf("Ingen flere\n");

  t = strrchr(txt, %s);
  printf("%%i %%s\n", (int) (t - txt), t + 1);
  return 0;
}
`, txt, tegnLiteral(c), tegnLiteral(c), tegnLiteral(d))

	return kode, ut.String(), []string{
		"strchr gir en peker til FØRSTE forekomst av tegnet (eller NULL), strrchr til SISTE.",
		"t - txt er indeksen pekeren står på.",
		"strchr(t + 1, ...) leter videre fra plassen etter den forrige.",
	}
}

func malSammenlign(r *rand.Rand) (string, string, []string) {
	txt := lagTekst(r)
	n := len(txt)

	// 1) !strncmp(txt + a, X, nx)
	nx := mellom(r, 2, 4)
	a := mellom(r, 0, n-nx)
	x := txt[a : a+nx]
	if r.IntN(2) == 0 {
		x = endre(r, x)
	}
	r1 := b2i(txt[a:a+nx] == x)

	// 2) strncmp(txt, Y, ny) == 0
	ny := mellom(r, 2, 5)
	y := txt[:ny]
	if r.IntN(2) == 0 {
		y = endre(r, y)
	}
	r2 := b2i(txt[:ny] == y)

	// 3) strstr(txt, Z)
	nz := mellom(r, 2, 3)
	zs := mellom(r, 0, n-nz)
	z := txt[zs : zs+nz]
	if r.IntN(3) == 0 {
		for i := 0; i < 10 && strings.Contains(txt, z); i++ {
			z = endre(r, z)
		}
	}
	idx := strings.Index(txt, z)
	k := mellom(r, 0, nz)

	// 4) strcmp(w1, w2) < 0 og strlen(w3)
	w1, w2 := velg(r, fornavn), velg(r, fornavn)
	if r.IntN(3) == 0 {
		w2 = strings.ToLower(w2)
	}
	w3 := velg(r, byer)

	var ut strings.Builder
	fmt.Fprintf(&ut, "%d %d\n", r1, r2)
	if idx >= 0 {
		fmt.Fprintf(&ut, "%d %s\n", idx, txt[idx+k:])
	} else {
		ut.WriteString("Fant ikke\n")
	}
	fmt.Fprintf(&ut, "%d %d\n", b2i(w1 < w2), len(w3))

	kode := fmt.Sprintf(`#include <stdio.h>
#include <string.h>

char txt[] = "%s";

int main()  {
  char* t;

  printf("%%i %%i\n", !strncmp(txt + %d, "%s", %d), strncmp(txt, "%s", %d) == 0);

  t = strstr(txt, "%s");
  if (t)  printf("%%i %%s\n", (int) (t - txt), t + %d);
  else    printf("Fant ikke\n");

  printf("%%i %%i\n", strcmp("%s", "%s") < 0, (int) strlen("%s"));
  return 0;
}
`, txt, a, x, nx, y, ny, z, k, w1, w2, w3)

	return kode, ut.String(), []string{
		"strncmp returnerer 0 når tekstene er LIKE – derfor er !strncmp(...) lik 1 ved likhet.",
		"strstr gir en peker til første forekomst av hele deltektsten, ellers NULL.",
		"I ASCII kommer alle store bokstaver før alle små ('Z' = 90 < 'a' = 97).",
	}
}

func malTegnaritmetikk(r *rand.Rand) (string, string, []string) {
	txt := lagTekst(r)
	n := len(txt)
	k := mellom(r, 2, 4)
	rr := mellom(r, 0, k-1)
	s := mellom(r, 0, n-4)
	e := mellom(r, s+3, min(n, s+6))
	d := mellom(r, 1, 3)
	bok := bokstavIndekser(txt)
	x, y, z := mellom(r, 0, n-1), mellom(r, 0, n-1), velg(r, bok)

	var ut strings.Builder
	for i := 0; i < n; i++ {
		if i%k == rr {
			ut.WriteByte(tilStor(txt[i]))
		} else {
			ut.WriteByte(tilLiten(txt[i]))
		}
	}
	ut.WriteString("\n")
	for i := s; i < e; i++ {
		ut.WriteByte(txt[i] + byte(d))
	}
	ut.WriteString("\n")
	fmt.Fprintf(&ut, "%d %d\n", int(txt[x])-int(txt[y]), int(tilStor(txt[z]))-int('A'))

	kode := fmt.Sprintf(`#include <stdio.h>
#include <ctype.h>

char txt[] = "%s";

int main()  {
  int i;

  for (i = 0;  txt[i];  i++)
      if (i %% %d == %d)  printf("%%c", toupper(txt[i]));
      else             printf("%%c", tolower(txt[i]));
  printf("\n");

  for (i = %d;  i < %d;  i++)
      printf("%%c", txt[i] + %d);
  printf("\n");

  printf("%%i %%i\n", txt[%d] - txt[%d], toupper(txt[%d]) - 'A');
  return 0;
}
`, txt, k, rr, s, e, d, x, y, z)

	return kode, ut.String(), []string{
		"toupper og tolower endrer bare bokstaver – '-' og mellomrom blir som de er.",
		"txt[i] + d skrevet med %c er tegnet d plasser lenger ut i ASCII-tabellen (vedlegg A).",
		"Tegn minus tegn er avstanden mellom dem i ASCII. toupper(c) - 'A' er plassen i alfabetet (A = 0).",
	}
}

func malByggTekst(r *rand.Rand) (string, string, []string) {
	txt := lagTekst(r)
	n := len(txt)
	k := mellom(r, 2, 4)
	a := mellom(r, 1, n-2)
	w := velg(r, []string{"!", "?", "123", "C", "xyz", "--", "OK"})
	ny2 := txt[a:] + w
	b := mellom(r, 1, len(ny2)-1)

	var rev []byte
	for i := n - 1; i >= 0; i -= k {
		rev = append(rev, txt[i])
	}

	var ut strings.Builder
	fmt.Fprintf(&ut, "%s %d\n", rev, len(rev))
	fmt.Fprintf(&ut, "%s\n", ny2)
	fmt.Fprintf(&ut, "%s %d\n", ny2[:b], b)

	kode := fmt.Sprintf(`#include <stdio.h>
#include <string.h>

char txt[] = "%s";

int main()  {
  char ny[80];
  int  i,  j = 0,  len = strlen(txt);

  for (i = len - 1;  i >= 0;  i -= %d)
      ny[j++] = txt[i];
  ny[j] = '\0';
  printf("%%s %%i\n", ny, j);

  strcpy(ny, txt + %d);
  strcat(ny, "%s");
  printf("%%s\n", ny);

  ny[%d] = '\0';
  printf("%%s %%i\n", ny, (int) strlen(ny));
  return 0;
}
`, txt, k, a, w, b)

	return kode, ut.String(), []string{
		"ny[j++] = ... lagrer på plass j og øker j etterpå – j blir antall tegn som er lagret.",
		"strcpy overskriver hele ny, strcat skjøter på slutten.",
		"ny[b] = '\\0' kutter teksten slik at den bare har b tegn.",
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
