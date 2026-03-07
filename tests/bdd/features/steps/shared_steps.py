"""Dynamically load step definitions from the checked-out insights-behavioral-spec repo.

This allows reusing common steps without duplicating them across repos.
If the repo was not checked out (continue-on-error in CI), this is a no-op.
"""

import importlib.util
import os
import sys

# Resolve relative to this file's location (tests/bdd/features/steps/ → project root)
_THIS_DIR = os.path.dirname(os.path.abspath(__file__))
_PROJECT_ROOT = os.path.normpath(os.path.join(_THIS_DIR, "..", "..", "..", ".."))
_SHARED_REPO_DIR = os.path.join(_PROJECT_ROOT, "insights-behavioral-spec")
_SHARED_STEPS_DIR = os.path.join(_SHARED_REPO_DIR, "features", "steps")

if os.path.isdir(_SHARED_STEPS_DIR):
    if _SHARED_REPO_DIR not in sys.path:
        sys.path.insert(0, _SHARED_REPO_DIR)
    if _SHARED_STEPS_DIR not in sys.path:
        sys.path.insert(0, _SHARED_STEPS_DIR)

    for _fname in sorted(os.listdir(_SHARED_STEPS_DIR)):
        if _fname.endswith(".py") and not _fname.startswith("_"):
            _fpath = os.path.join(_SHARED_STEPS_DIR, _fname)
            _mod_name = f"_shared_steps_{_fname[:-3]}"
            _spec = importlib.util.spec_from_file_location(_mod_name, _fpath)
            _mod = importlib.util.module_from_spec(_spec)
            sys.modules[_mod_name] = _mod
            _spec.loader.exec_module(_mod)