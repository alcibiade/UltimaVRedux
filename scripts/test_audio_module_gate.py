"""Regression checks for the staged consumer's narrowly scoped substitutions."""

import copy
import unittest

from audio_module_gate import MODULE, ROOT, json_objects, verify_bridge_replacements


class AudioModuleGateTests(unittest.TestCase):
    def metadata(self):
        return {"Replace": [
            {"Old": {"Path": MODULE}, "New": {"Path": "../library"}},
            {"Old": {"Path": "github.com/bradhannah/UltimaVRedux"}, "New": {"Path": str(ROOT)}},
        ]}

    def test_only_known_test_replacements_are_accepted(self):
        verify_bridge_replacements(self.metadata())
        for change in ("extra", "version", "outside", "duplicate", "missing"):
            with self.subTest(change=change):
                metadata = copy.deepcopy(self.metadata())
                if change == "extra":
                    metadata["Replace"].append({"Old": {"Path": "example.com/other"}, "New": {"Path": "../other"}})
                elif change == "version":
                    metadata["Replace"][0]["Old"]["Version"] = "v1.0.0"
                elif change == "outside":
                    metadata["Replace"][0]["New"]["Path"] = "../../other"
                elif change == "duplicate":
                    metadata["Replace"].append(metadata["Replace"][0])
                else:
                    metadata["Replace"].pop()
                with self.assertRaises(ValueError):
                    verify_bridge_replacements(metadata)

    def test_dependency_stream_decodes_all_objects(self):
        self.assertEqual(list(json_objects(' {"a": 1}\n {"b": 2} \n')), [{"a": 1}, {"b": 2}])
