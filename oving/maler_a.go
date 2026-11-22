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
	k5 := mellom(r, 1, 3)

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
	fmt.Fprintf(&ut, "%c%c%c\n", txt[0], txt[n/2], txt[n-k5])

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
  printf("%%c%%c%%c\n", txt[0], txt[len / 2], txt[len - %d]);
  return 0;
}
`, txt, start, steg, steg2, m, m2, k5)

	return kode, ut.String(), []string{
		"Skriv indeksene under teksten (0, 1, 2, …) og list opp alle verdiene i tar i hver løkke.",
		"txt + m er teksten fra indeks m og ut – printf(\"%s\", txt + m) skriver resten av teksten.",
		"strlen teller ikke med '\\0'. len / 2 er heltallsdivisjon.",
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

	// Linje 4: gå baklengs fra p2 til tegnet c3 (som garantert finnes før p2).
	p2 := mellom(r, 1, n-2)
	var foran []int
	for _, i := range bok {
		if i <= p2 {
			foran = append(foran, i)
		}
	}
	c3 := txt[velg(r, foran)]
	j := strings.LastIndexByte(txt[:p2+1], c3)

	// Linje 5: *t++, *++t og ++*t.
	q := mellom(r, 0, n-3)

	var ut strings.Builder
	fmt.Fprintf(&ut, "%d %c\n", tell, txt[n-k])
	fmt.Fprintf(&ut, "%d %s\n", steg, rest)
	fmt.Fprintf(&ut, "%c%c%c\n", txt[a], txt[b], txt[c]+1)
	fmt.Fprintf(&ut, "%d %c\n", j, txt[j+1])
	fmt.Fprintf(&ut, "%c%c%c\n", txt[q], txt[q+2], txt[q+2]+1)

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

  t = txt + %d;
  while (*t != %s)  t--;
  printf("%%i %%c\n", (int) (t - txt), *(t + 1));

  t = txt + %d;
  printf("%%c", *t++);
  printf("%%c", *++t);
  printf("%%c\n", ++*t);
  return 0;
}
`, txt, tegnLiteral(c1), k, p, tegnLiteral(c2), a, b, c, p2, tegnLiteral(c3), q)

	return kode, ut.String(), []string{
		"*t++ == 'x' sammenligner tegnet der t står, og flytter t etterpå – også når sammenligningen er usann.",
		"Etter while (*t) står t på '\\0', altså indeks strlen(txt). *(t - k) er da k plasser fra slutten.",
		"Finnes ikke tegnet, stopper den andre løkka på '\\0', og %s skriver en tom tekst.",
		"*(txt + c) + 1 er tegnet rett etter i ASCII-tabellen (kapittel 6 i bibelen).",
		"*t++ bruker tegnet og flytter t etterpå, *++t flytter først, og ++*t øker selve TEGNET (ikke pekeren).",
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

	e := txt[velg(r, bok)]
	if r.IntN(4) == 0 {
		e = ikkeIBruk(r, txt)
	}
	antE := strings.Count(txt, string(e))
	sep := skilletegn(txt)
	forsteOrd := txt[:strings.IndexByte(txt, sep)]

	var ut strings.Builder
	fmt.Fprintf(&ut, "%d %c%c\n", forste, txt[forste-1], txt[forste+1])
	if neste := strings.IndexByte(txt[forste+1:], c); neste >= 0 {
		fmt.Fprintf(&ut, "%s\n", txt[forste+1+neste:])
	} else {
		ut.WriteString("Ingen flere\n")
	}
	fmt.Fprintf(&ut, "%d %s\n", siste, txt[siste+1:])
	fmt.Fprintf(&ut, "%d\n", antE)
	fmt.Fprintf(&ut, "%s %d\n", forsteOrd, len(forsteOrd))

	kode := fmt.Sprintf(`#include <stdio.h>
#include <string.h>

char txt[] = "%s";

int main()  {
  char* t = strchr(txt, %s);
  int   n = 0;

  printf("%%i %%c%%c\n", (int) (t - txt), *(t - 1), *(t + 1));

  t = strchr(t + 1, %s);
  if (t)  printf("%%s\n", t);
  else    printf("Ingen flere\n");

  t = strrchr(txt, %s);
  printf("%%i %%s\n", (int) (t - txt), t + 1);

  t = strchr(txt, %s);
  while (t)  {  n++;  t = strchr(t + 1, %s);  }
  printf("%%i\n", n);

  t = strchr(txt, %s);
  *t = '\0';
  printf("%%s %%i\n", txt, (int) strlen(txt));
  return 0;
}
`, txt, tegnLiteral(c), tegnLiteral(c), tegnLiteral(d), tegnLiteral(e), tegnLiteral(e), tegnLiteral(sep))

	return kode, ut.String(), []string{
		"strchr gir en peker til FØRSTE forekomst av tegnet (eller NULL), strrchr til SISTE.",
		"t - txt er indeksen pekeren står på.",
		"strchr(t + 1, ...) leter videre fra plassen etter den forrige – slik kan en løkke telle alle forekomster.",
		"*t = '\\0' kutter selve teksten der t peker, så txt slutter der.",
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

	// 4) strcmp(txt, w4) > 0 og !strcmp(txt + x4, s4)
	w4 := velg(r, byer)
	x4 := mellom(r, 1, n-1)
	s4 := txt[x4:]
	if r.IntN(2) == 0 {
		s4 = endre(r, s4)
	}

	// 5) strncmp(v1, v2, n5) == 0 der v2 er v1 med en endring et sted bak,
	//    og strlen(txt + y5)
	v1 := velg(r, fornavn)
	kk := mellom(r, 1, len(v1)-1)
	v2 := v1[:kk] + endre(r, v1[kk:])
	n5 := mellom(r, 1, len(v1))
	y5 := mellom(r, 0, n)

	var ut strings.Builder
	fmt.Fprintf(&ut, "%d %d\n", r1, r2)
	if idx >= 0 {
		fmt.Fprintf(&ut, "%d %s\n", idx, txt[idx+k:])
	} else {
		ut.WriteString("Fant ikke\n")
	}
	fmt.Fprintf(&ut, "%d %d\n", b2i(w1 < w2), len(w3))
	fmt.Fprintf(&ut, "%d %d\n", b2i(txt > w4), b2i(txt[x4:] == s4))
	fmt.Fprintf(&ut, "%d %d\n", b2i(v1[:n5] == v2[:n5]), n-y5)

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
  printf("%%i %%i\n", strcmp(txt, "%s") > 0, !strcmp(txt + %d, "%s"));
  printf("%%i %%i\n", strncmp("%s", "%s", %d) == 0, (int) strlen(txt + %d));
  return 0;
}
`, txt, a, x, nx, y, ny, z, k, w1, w2, w3, w4, x4, s4, v1, v2, n5, y5)

	return kode, ut.String(), []string{
		"strncmp returnerer 0 når tekstene er LIKE – derfor er !strncmp(...) lik 1 ved likhet.",
		"strstr gir en peker til første forekomst av hele deltektsten, ellers NULL.",
		"I ASCII kommer alle store bokstaver før alle små ('Z' = 90 < 'a' = 97).",
		"strncmp(a, b, n) ser bare på de n første tegnene – er forskjellen lenger bak, regnes de som like.",
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
	x5, y5 := mellom(r, 0, n-1), velg(r, bok)

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
	for i := 0; i < n; i++ {
		if txt[i] >= 'A' && txt[i] <= 'Z' {
			ut.WriteByte(txt[i] + 32)
		}
	}
	ut.WriteString("\n")
	fmt.Fprintf(&ut, "%d %c\n", int(txt[x5]), txt[y5]-1)

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

  for (i = 0;  txt[i];  i++)
      if (isupper(txt[i]))  printf("%%c", txt[i] + 32);
  printf("\n");

  printf("%%i %%c\n", txt[%d], txt[%d] - 1);
  return 0;
}
`, txt, k, rr, s, e, d, x, y, z, x5, y5)

	return kode, ut.String(), []string{
		"toupper og tolower endrer bare bokstaver – '-' og mellomrom blir som de er.",
		"txt[i] + d skrevet med %c er tegnet d plasser lenger ut i ASCII-tabellen (kapittel 6 i bibelen).",
		"Tegn minus tegn er avstanden mellom dem i ASCII. toupper(c) - 'A' er plassen i alfabetet (A = 0).",
		"Stor bokstav + 32 er tilsvarende liten bokstav. %i skriver ASCII-koden til et tegn.",
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
	c6 := mellom(r, 0, n-1)
	ny3 := ny2[:b] + txt[c6:]
	sep := skilletegn(txt)
	init := []byte{txt[0]}
	for i := 1; i < n; i++ {
		if txt[i-1] == sep {
			init = append(init, txt[i])
		}
	}

	var rev []byte
	for i := n - 1; i >= 0; i -= k {
		rev = append(rev, txt[i])
	}

	var ut strings.Builder
	fmt.Fprintf(&ut, "%s %d\n", rev, len(rev))
	fmt.Fprintf(&ut, "%s\n", ny2)
	fmt.Fprintf(&ut, "%s %d\n", ny2[:b], b)
	fmt.Fprintf(&ut, "%s\n", ny3)
	fmt.Fprintf(&ut, "%s %d\n", init, len(init))

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

  strcat(ny, txt + %d);
  printf("%%s\n", ny);

  j = 0;
  ny[j++] = txt[0];
  for (i = 1;  txt[i];  i++)
      if (txt[i - 1] == %s)  ny[j++] = txt[i];
  ny[j] = '\0';
  printf("%%s %%i\n", ny, j);
  return 0;
}
`, txt, k, a, w, b, c6, tegnLiteral(sep))

	return kode, ut.String(), []string{
		"ny[j++] = ... lagrer på plass j og øker j etterpå – j blir antall tegn som er lagret.",
		"strcpy overskriver hele ny, strcat skjøter på slutten.",
		"ny[b] = '\\0' kutter teksten slik at den bare har b tegn – en senere strcat skjøter på derfra.",
		"Siste løkke tar med første tegn og hvert tegn som står rett etter et skilletegn.",
	}
}

// skilletegn gir tegnet lagTekst brukte mellom bynavnene ('-' eller ' ').
func skilletegn(txt string) byte {
	if strings.IndexByte(txt, '-') >= 0 {
		return '-'
	}
	return ' '
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
