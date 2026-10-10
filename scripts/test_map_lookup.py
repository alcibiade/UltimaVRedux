"""Original-data discovery uses explicit configuration without personal paths."""

import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location("map_lookup", Path(__file__).with_name("map-lookup.py"))
LOOKUP = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(LOOKUP)


class MapLookupTests(unittest.TestCase):
    def test_explicit_data_directory_takes_precedence_and_never_silently_falls_back(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "OLD").mkdir()
            (root / "OLD/BRIT.DAT").touch()
            explicit = root / "explicit"
            explicit.mkdir()
            with patch.object(LOOKUP, "find_script_dir", return_value=root), \
                 patch.dict(LOOKUP.os.environ, {"ULTIMA5_DATA_DIR": str(explicit)}, clear=True):
                self.assertIsNone(LOOKUP.find_data_path())
                (explicit / "BRIT.DAT").touch()
                self.assertEqual(explicit, LOOKUP.find_data_path())
                with patch.dict(LOOKUP.os.environ, {"ULTIMA5_DATA_PATH": str(root / "missing")}):
                    self.assertIsNone(LOOKUP.find_data_path())
                with patch.dict(LOOKUP.os.environ, {"ULTIMA5_DATA_PATH": str(root / "OLD")}):
                    self.assertEqual(root / "OLD", LOOKUP.find_data_path())
            with patch.object(LOOKUP, "find_script_dir", return_value=root), \
                 patch.dict(LOOKUP.os.environ, {}, clear=True):
                self.assertEqual(root / "OLD", LOOKUP.find_data_path())


if __name__ == "__main__":
    unittest.main()
