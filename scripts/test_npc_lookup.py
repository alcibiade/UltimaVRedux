"""Asset-independent regressions for semantic NPC lookup snapshots."""

import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("npc_lookup", Path(__file__).with_name("npc-lookup.py"))
assert spec is not None and spec.loader is not None
lookup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(lookup)


class SemanticLookupTests(unittest.TestCase):
    def setUp(self):
        self.documents = [{
            "id": "original.towne.007",
            "origin": {"kind": "original", "source": "towne", "dialog": 7},
            "name": [{"say": [{"text": "Mara "}, {"runic": "Keeper"}]}],
            "topics": [{"keywords": ["WORD", "POWE"], "response": [{"say": [{"text": "The word is "}, {"runic": "EXAMPLE"}]}]}],
            "questions": {"question_01": {"answers": [{"keywords": ["DAWN"], "response": "Confirmed."}]}},
        }]

    def test_name_lookup_joins_source_and_one_based_dialog_to_real_npc_slots(self):
        npcs = {"npcs": [
            {"location": 17, "dialog_number": 7},  # Same number, wrong family.
            {"location": 2, "dialog_number": 6},   # Off-by-one neighbor.
            {"location": 2, "dialog_number": 7},
            {"location": 3, "dialog_number": 7},
        ]}
        with patch.object(lookup, "load_conversations", return_value=self.documents), \
             patch.object(lookup, "load_npc_references", return_value=npcs):
            matches = lookup.find_npc_by_name("mara")
            self.assertEqual([2, 3], [match["npc_index"] for match in matches])
            self.assertEqual([2], [match["npc_index"] for match in lookup.find_npc_by_name("mara", 2)])
            self.assertEqual("Mara Keeper", matches[0]["matched_text"])
            self.assertEqual(7, matches[0]["dialog_number"])

    def test_keywords_include_topics_and_question_answers_with_runic_preview(self):
        with patch.object(lookup, "load_conversations", return_value=self.documents):
            matches = lookup.find_dialog_by_keyword("word")
            self.assertEqual(1, len(matches))
            self.assertEqual("The word is EXAMPLE", matches[0]["response_preview"])
            self.assertEqual("original.towne.007", matches[0]["semantic_id"])
            self.assertEqual("Confirmed.", lookup.find_dialog_by_keyword("dawn")[0]["response_preview"])
            self.assertEqual([], lookup.find_dialog_by_keyword("hidden"))

    def test_preview_preserves_breaks_and_omits_control_actions(self):
        self.assertEqual("First\nSecond", lookup.display_text([
            {"say": [{"text": "First"}, "line_break", {"runic": "Second"}]},
            "pause", {"ask": "question_01"}, "return_to_topics",
        ]))


if __name__ == "__main__":
    unittest.main()
