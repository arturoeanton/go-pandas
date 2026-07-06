#!/usr/bin/env python3
"""Regenerate every golden file from real pandas/NumPy, then run the Go
compatibility tests against them.

    python3 compat/python/run_compat_suite.py

Requires python3 with pandas+numpy, and the Go toolchain.
"""
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", ".."))


def run(cmd, **kwargs):
    print("+", " ".join(cmd))
    subprocess.run(cmd, check=True, **kwargs)


def main():
    run([sys.executable, os.path.join(HERE, "generate_pandas_goldens.py")])
    run([sys.executable, os.path.join(HERE, "generate_numpy_goldens.py")])
    run(["go", "test", "./compat/..."], cwd=ROOT)
    print("compat suite OK")


if __name__ == "__main__":
    main()
