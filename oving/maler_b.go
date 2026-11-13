package main

// Del B: struct, funksjoner og parametre (som oppgave 1b på eksamen).

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

func malStructKopi(r *rand.Rand) (string, string, []string) {
	type person struct {
		navn  string
		alder int
	}
	n1 := velg(r, fornavn)
	n2 := velg(r, fornavn)
	for n2 == n1 {
		n2 = velg(r, fornavn)
	}
	alderA, alderB := mellom(r, 15, 60), mellom(r, 15, 60)
	a := person{n1, alderA}
	b := person{n2, alderB}
	x1, x2 := mellom(r, 1, 9), mellom(r, 1, 9)
	s1 := velg(r, []string{"!", "+", "#", "x"})
	s2 := velg(r, []string{"?", "*", "@", "y"})

	// 3 runder med 1-2 operasjoner. Maks én tilordning (a = b eller b = a).
	ops := []string{"funk1(a);", "funk1(b);", "funk2(&a);", "funk2(&b);", "tilordning"}
	var linjer []string
	var ut strings.Builder
	brukt := false
	for runde := 0; runde < 3; runde++ {
		for i, antall := 0, mellom(r, 1, 2); i < antall; i++ {
			op := velg(r, ops)
			if op == "tilordning" && brukt {
				op = velg(r, ops[:4])
			}
			switch op {
			case "funk2(&a);":
				a.alder += x2
				a.navn += s2
			case "funk2(&b);":
				b.alder += x2
				b.navn += s2
			case "tilordning":
				brukt = true
				if r.IntN(2) == 0 {
					a, op = b, "a = b;"
				} else {
					b, op = a, "b = a;"
				}
			}
			// funk1 får en kopi: ingen endring hos kalleren.
			linjer = append(linjer, "  "+op)
		}
		linjer = append(linjer, `  printf("%s %i - %s %i\n", a.navn, a.alder, b.navn, b.alder);`)
		fmt.Fprintf(&ut, "%s %d - %s %d\n", a.navn, a.alder, b.navn, b.alder)
		if runde < 2 {
			linjer = append(linjer, "")
		}
	}
	l1, l2 := mellom(r, 3, 7), mellom(r, 3, 7)
	fmt.Fprintf(&ut, "%d %d\n", b2i(len(a.navn) > l1), b2i(len(b.navn) > l2))

	kode := fmt.Sprintf(`#include <stdio.h>
#include <string.h>
#include <stdbool.h>

struct Person  {  char navn[30];  int alder;  };

void funk1(struct Person p)   {  p.alder += %d;   strcat(p.navn, "%s");  }
void funk2(struct Person* p)  {  p->alder += %d;  strcat(p->navn, "%s");  }
bool funk3(const struct Person* p, const int n)  {  return ((int) strlen(p->navn) > n);  }

int main()  {
  struct Person a = { "%s", %d },  b = { "%s", %d };

%s

  printf("%%i %%i\n", funk3(&a, %d), funk3(&b, %d));
  return 0;
}
`, x1, s1, x2, s2, n1, alderA, n2, alderB, strings.Join(linjer, "\n"), l1, l2)

	return kode, ut.String(), []string{
		"funk1 får en KOPI av structen – endringene (også strcat) forsvinner når funksjonen er ferdig.",
		"funk2 får ADRESSEN – endringene blir stående i originalen.",
		"a = b kopierer hele innholdet, også char-arrayen navn.",
		"bool skrives ut som 1 eller 0 med %i.",
	}
}

func malBytt(r *rand.Rand) (string, string, []string) {
	navn := []string{"x", "y", "z"}
	start := [3]int{mellom(r, 1, 20), mellom(r, 1, 20), mellom(r, 1, 20)}
	v := start

	var linjer []string
	var ut strings.Builder
	for kall := 1; kall <= 6; kall++ {
		i := r.IntN(3)
		j := (i + 1 + r.IntN(2)) % 3 // j != i
		switch r.IntN(3) {
		case 0:
			linjer = append(linjer, fmt.Sprintf("  bytt1(%s, %s);", navn[i], navn[j]))
		case 1:
			v[i], v[j] = v[j], v[i]
			linjer = append(linjer, fmt.Sprintf("  bytt2(&%s, &%s);", navn[i], navn[j]))
		case 2:
			v[i] *= 2
			linjer = append(linjer, fmt.Sprintf("  dobbel(&%s, %s);", navn[i], navn[j]))
		}
		if kall%2 == 0 {
			linjer = append(linjer, `  printf("%i %i %i\n", x, y, z);`)
			fmt.Fprintf(&ut, "%d %d %d\n", v[0], v[1], v[2])
			if kall < 6 {
				linjer = append(linjer, "")
			}
		}
	}

	kode := fmt.Sprintf(`#include <stdio.h>

void bytt1(int a, int b)    {  int t = a;   a = b;    b = t;  }
void bytt2(int* a, int* b)  {  int t = *a;  *a = *b;  *b = t;  }
void dobbel(int* a, int b)  {  *a *= 2;     b *= 2;  }

int main()  {
  int x = %d,  y = %d,  z = %d;

%s
  return 0;
}
`, start[0], start[1], start[2], strings.Join(linjer, "\n"))

	return kode, ut.String(), []string{
		"bytt1 får KOPIER av tallene – ingenting endres hos kalleren.",
		"Bare parametre med * (og & i kallet) kan endre kallerens variable.",
		"I dobbel(&x, y) dobles x, mens kopien av y dobles og forsvinner.",
	}
}

func malVarer(r *rand.Rand) (string, string, []string) {
	type vare struct {
		navn   string
		antall int
		pris   float32
	}
	// Priser er hele eller halve kroner, så alle summer er eksakte i float.
	var vs [4]vare
	brukt := map[string]bool{}
	for i := range vs {
		n := velg(r, varer)
		for brukt[n] {
			n = velg(r, varer)
		}
		brukt[n] = true
		vs[i] = vare{n, mellom(r, 0, 9), float32(mellom(r, 8, 80)) / 2}
	}
	start := vs

	i1, n1 := r.IntN(4), mellom(r, 1, 6)
	i2, n2 := r.IntN(4), mellom(r, 1, 6)
	ir1, ir2 := r.IntN(4), r.IntN(4)
	grense := float32(mellom(r, 10, 30))

	kjop := func(i, n int) {
		vs[i].antall -= n
		if vs[i].antall < 0 {
			vs[i].antall = 0
		}
	}
	kjop(i1, n1)
	// vareRabatt(gVarer[ir1]) får en kopi: ingen endring.
	vs[ir2].pris /= 2 // vareRabatt2(&gVarer[ir2])
	kjop(i2, n2)

	var total float32
	dyre := 0
	for i := 0; i < 4; i++ {
		total += float32(vs[i].antall) * vs[i].pris
		if vs[i].pris > grense {
			dyre++
		}
	}
	var ut strings.Builder
	fmt.Fprintf(&ut, "%.2f %d\n", float64(total), dyre)
	for i := 3; i >= 0; i-- {
		if vs[i].antall > 0 {
			fmt.Fprintf(&ut, "%s:%d ", vs[i].navn, vs[i].antall)
		}
	}
	ut.WriteString("\n")

	var init []string
	for _, v := range start {
		init = append(init, fmt.Sprintf(`{ "%s", %d, %s }`, v.navn, v.antall, prisLiteral(v.pris)))
	}

	kode := fmt.Sprintf(`#include <stdio.h>
#include <stdbool.h>

struct Vare  {  char navn[20];  int antall;  float pris;  };

struct Vare gVarer[] = { %s,
                         %s };
const int ANTVARER = 4;

float vareSum(const struct Vare* v)                {  return v->antall * v->pris;  }
bool  vareDyr(const struct Vare v, const float g)  {  return (v.pris > g);  }
void  vareRabatt(struct Vare v)                    {  v.pris /= 2;  }
void  vareRabatt2(struct Vare* v)                  {  v->pris /= 2;  }
void  vareKjop(struct Vare* v, const int n)  {
  v->antall -= n;
  if (v->antall < 0)  v->antall = 0;
}

int main()  {
  int   i,  dyre = 0;
  float total = 0;

  vareKjop(&gVarer[%d], %d);
  vareRabatt(gVarer[%d]);
  vareRabatt2(&gVarer[%d]);
  vareKjop(&gVarer[%d], %d);

  for (i = 0;  i < ANTVARER;  i++)  {
    total += vareSum(&gVarer[i]);
    if (vareDyr(gVarer[i], %s))  dyre++;
  }
  printf("%%.2f %%i\n", total, dyre);

  for (i = ANTVARER - 1;  i >= 0;  i--)
    if (gVarer[i].antall > 0)  printf("%%s:%%i ", gVarer[i].navn, gVarer[i].antall);
  printf("\n");
  return 0;
}
`, strings.Join(init[:2], ", "), strings.Join(init[2:], ", "),
		i1, n1, ir1, ir2, i2, n2, prisLiteral(grense))

	return kode, ut.String(), []string{
		"vareRabatt får en KOPI – prisen i arrayen endres ikke. vareRabatt2 får adressen og halverer prisen.",
		"vareKjop setter antallet til 0 hvis det ville blitt negativt.",
		"%.2f skriver alltid to desimaler. Siste linje har et mellomrom etter hver vare.",
	}
}

// prisLiteral skriver en float som C-konstant med én desimal (f.eks. 12.5).
func prisLiteral(p float32) string { return fmt.Sprintf("%.1f", p) }

func malTallrekke(r *rand.Rand) (string, string, []string) {
	mult := velg(r, []int{3, 3, 5})
	add := velg(r, []int{1, 1, 3})
	n0 := mellom(r, 3, 30)
	n := n0
	k := mellom(r, 5, 8)
	bredde := velg(r, []int{3, 4, 5})
	d := mellom(r, 2, 4)

	var ut strings.Builder
	sum, maks := 0, 0
	for i := 1; i <= k; i++ {
		if n%2 == 0 {
			n = n / 2
		} else {
			n = mult*n + add
		}
		sum += n
		if n > maks {
			maks = n
		}
		fmt.Fprintf(&ut, "%*d", bredde, n)
	}
	fmt.Fprintf(&ut, "\n%d %d %d\n", maks, sum/d, sum%d)
	fmt.Fprintf(&ut, "%.1f\n", float64(float32(sum)/float32(k)))

	kode := fmt.Sprintf(`#include <stdio.h>

int neste(const int n)  {
  if (n %% 2 == 0)  return n / 2;
  else             return %d * n + %d;
}

int main()  {
  int i,  n = %d,  sum = 0,  maks = 0;

  for (i = 1;  i <= %d;  i++)  {
    n = neste(n);
    sum += n;
    if (n > maks)  maks = n;
    printf("%%%di", n);
  }
  printf("\n%%i %%i %%i\n", maks, sum / %d, sum %% %d);
  printf("%%.1f\n", (float) sum / %d);
  return 0;
}
`, mult, add, n0, k, bredde, d, d, k)

	return kode, ut.String(), []string{
		"Følg n, sum og maks i en tabell – én rad per runde i løkka.",
		fmt.Sprintf("%%%di skriver tallet høyrejustert med bredde %d (mellomrom foran).", bredde, bredde),
		"sum / d og sum % d er heltallsdivisjon og rest. (float) sum / k gir desimaler, og %.1f runder til én desimal.",
	}
}
