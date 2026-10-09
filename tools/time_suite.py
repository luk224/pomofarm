#!/usr/bin/env python3
"""Ejecuta la suite de pruebas de tiempo descrita en docs/qa/tiempo.md (P5-03).

Lee los nombres de pruebas Go (`TestXxx`) y de scripts E2E (`pN_NN_xxx.py`) que aparecen entre comillas invertidas en el documento,
comprueba que TODOS existen (si el documento nombra una prueba que ya no existe, falla) y ejecuta las pruebas Go. Con --e2e ejecuta
además los scripts de navegador con tools/e2e/run_isolated.sh.

Uso: tools/time_suite.py [--e2e]
"""
import re, subprocess, sys
from pathlib import Path

root = Path(__file__).resolve().parent.parent
doc = (root / "docs/qa/tiempo.md").read_text(encoding="utf-8")
names = sorted(set(re.findall(r"`(Test[A-Za-z0-9_]+)`", doc)))
scripts = sorted(set(re.findall(r"`(p\d_\d+[a-z]?_[a-z0-9_]+\.py)`", doc)))

# 1) cada prueba nombrada existe
declared = set()
for f in (root / "backend").rglob("*_test.go"):
    declared |= set(re.findall(r"^func (Test[A-Za-z0-9_]+)\(", f.read_text(encoding="utf-8"), re.M))
missing = [n for n in names if n not in declared]
missing += [s for s in scripts if not (root / "tools/e2e" / s).exists()]
if missing:
    sys.exit("El documento nombra pruebas que no existen: " + ", ".join(missing))
print(f"{len(names)} pruebas Go y {len(scripts)} scripts E2E nombrados, todos existen.")

# 2) ejecutar las pruebas Go (todas, en todos los paquetes: -run solo filtra por nombre)
pattern = "^(" + "|".join(names) + ")$"
r = subprocess.run(["go", "test", "./...", "-count=1", "-run", pattern, "-v"], cwd=root / "backend", capture_output=True, text=True)
passed = len(re.findall(r"^--- PASS", r.stdout, re.M)); failed = re.findall(r"^--- FAIL: (\S+)", r.stdout, re.M); skipped = re.findall(r"^--- SKIP: (\S+)", r.stdout, re.M)
ran = set(re.findall(r"^--- (?:PASS|FAIL|SKIP): (\S+)", r.stdout, re.M))
print(f"Go: {passed} pasan, {len(failed)} fallan, {len(skipped)} omitidas")
never = [n for n in names if n not in ran]
if never:
    print("Pruebas nombradas que NO se ejecutaron:", ", ".join(never)); r.returncode = r.returncode or 1
if failed:
    print("Fallan:", ", ".join(failed)); print(r.stdout[-3000:])
if r.returncode:
    sys.exit(1)

# 3) navegador
if "--e2e" in sys.argv:
    cmd = [str(root / "tools/e2e/run_isolated.sh")] + [str(root / "tools/e2e" / s) for s in scripts]
    e = subprocess.run(cmd, cwd=root)
    if e.returncode:
        sys.exit(e.returncode)
print("Suite de tiempo: todo en orden.")
