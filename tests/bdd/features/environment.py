"""Forward behave environment hooks to insights-behavioral-spec/features/environment.py.

Behave looks for environment.py in the features directory relative to its cwd
(tests/bdd/features/). This file finds the shared environment.py using a path
relative to __file__, so it works regardless of the working directory.
"""

import importlib.util
import os

_THIS_DIR = os.path.dirname(os.path.abspath(__file__))
_PROJECT_ROOT = os.path.normpath(os.path.join(_THIS_DIR, "..", "..", ".."))
_SHARED_ENV = os.path.join(
    _PROJECT_ROOT, "insights-behavioral-spec", "features", "environment.py"
)

if os.path.isfile(_SHARED_ENV):
    _spec = importlib.util.spec_from_file_location("_shared_environment", _SHARED_ENV)
    _mod = importlib.util.module_from_spec(_spec)
    _spec.loader.exec_module(_mod)

    before_all = getattr(_mod, "before_all", None)
    before_feature = getattr(_mod, "before_feature", None)
    before_scenario = getattr(_mod, "before_scenario", None)
    after_scenario = getattr(_mod, "after_scenario", None)
