"""Forward behave environment hooks to insights-behavioral-spec/features/environment.py.

Behave looks for environment.py in the features directory relative to its cwd
(tests/bdd/features/). This file finds the shared environment.py using a path
relative to __file__, so it works regardless of the working directory.
"""

import importlib.util
import os
import sys


def _load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    sys.modules[name] = mod
    spec.loader.exec_module(mod)
    return mod


_THIS_DIR = os.path.dirname(os.path.abspath(__file__))
_PROJECT_ROOT = os.path.normpath(os.path.join(_THIS_DIR, "..", "..", ".."))
_SHARED_FEATURES_DIR = os.path.join(
    _PROJECT_ROOT, "insights-behavioral-spec", "features"
)
_SHARED_ENV = os.path.join(_SHARED_FEATURES_DIR, "environment.py")
_SHARED_ENV_TEST = os.path.join(_SHARED_FEATURES_DIR, "environment_test.py")

if os.path.isfile(_SHARED_ENV):
    _mod = _load_module("environment", _SHARED_ENV)

    before_all = getattr(_mod, "before_all", None)
    before_feature = getattr(_mod, "before_feature", None)
    before_scenario = getattr(_mod, "before_scenario", None)
    after_scenario = getattr(_mod, "after_scenario", None)

if os.path.isfile(_SHARED_ENV_TEST):
    _load_module("_shared_environment_test", _SHARED_ENV_TEST)
