"""Public-only schema/graph and application-locator regressions."""

import json
import tempfile
import unittest
from pathlib import Path

import gamefacts_public as facts


def rule(identifier="RULE-0001", uses="none", kind="rule"):
    text = f"### {identifier}\n\n**Kind:** {kind}\n**Uses:** {uses}\n"
    if kind == "rule":
        text += "**Original:** Incomplete\n**Redux:** Unverified\n"
        text += "**Implementation:** `internal/sample.go`\n**Tests:** `internal/sample_test.go`\n"
    return text + "\nBehavior and limits.\n"


class PublicFactsTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.document = "docs/gamefacts/RULES.md"
        self.write(self.document, "# Scope\n\n" + rule())
        self.write("docs/gamefacts/catalog.json", json.dumps({"schema_version": 1, "documents": [self.document]}))
        self.write("internal/sample.go", "package sample\n// ref:RULE-0001.\n")
        self.write("internal/sample_test.go", "package sample\n")

    def write(self, path, text):
        destination = self.root / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(text, encoding="utf-8")

    def test_standalone_checkout_needs_only_public_files(self):
        documents, rules = facts.validate_repository(self.root)
        self.assertEqual([self.document], list(documents))
        self.assertEqual(["RULE-0001"], list(rules))

    def test_sections_and_preamble_are_verbatim_for_review_hashing(self):
        preamble = "# Scope\n\n```markdown\n### EXAMPLE-0999\n```\n"
        first, second = rule(), rule("RULE-0002", "RULE-0001")
        prefix, rules = facts.parse_public(preamble + first + second)
        self.assertEqual(preamble, prefix)
        self.assertEqual(first, rules["RULE-0001"]["text"])
        self.assertEqual(second, rules["RULE-0002"]["text"])

    def test_graph_failures(self):
        cases = [
            (rule() + rule(), "duplicate public ID"),
            (rule(uses="RULE-9999"), "missing dependency"),
            (rule(uses="broken"), "malformed dependency ID"),
            (rule(uses="RULE-0001"), "dependency cycle"),
            (rule() + rule("RULE-0002", "RULE-0001", "group") + rule("RULE-0003", "RULE-0002"), "leaf rules"),
            (rule() + "\n[missing](#rule-9999)\n", "missing public anchor"),
            (rule().replace("Incomplete", "Assumed"), "invalid Original"),
            (rule().replace("Unverified", "Done"), "invalid Redux"),
            (rule().replace("**Kind:** rule", "**Kind:** rule\n**Kind:** rule"), "duplicate Kind"),
            (rule() + rule("rule-0002"), "malformed public ID heading"),
            (rule() + rule("RULE-00012"), "malformed public ID heading"),
        ]
        for text, expected in cases:
            with self.subTest(expected=expected):
                with self.assertRaisesRegex(ValueError, expected):
                    facts.parse_public(text)

    def test_expansion_is_bounded_and_deduplicated(self):
        _, rules = facts.parse_public(rule() + rule("RULE-0002", "RULE-0001") +
                                      rule("RULE-0003", "RULE-0001, RULE-0002", "group"))
        self.assertEqual(["RULE-0001", "RULE-0002", "RULE-0003"], facts.expand(rules, "RULE-0003"))
        text = rule() + "".join(rule(f"RULE-{index:04}", f"RULE-{index-1:04}")
                                for index in range(2, facts.MAX_DEPTH + 3))
        with self.assertRaisesRegex(ValueError, "depth exceeds"):
            facts.parse_public(text)

    def test_catalog_cross_topic_graph_and_links(self):
        second = "docs/gamefacts/OTHER.md"
        self.write(second, rule("OTHER-0001", "RULE-0001"))
        self.write("docs/gamefacts/catalog.json", json.dumps({"schema_version": 1, "documents": [self.document, second]}))
        self.write(self.document, rule() + "\n[other](OTHER.md#other-0001)\n")
        _, rules = facts.validate_repository(self.root)
        self.assertEqual(["RULE-0001", "OTHER-0001"], facts.expand(rules, "OTHER-0001"))
        self.write(self.document, rule() + "\n[wrong](OTHER.md#rule-0001)\n")
        with self.assertRaisesRegex(ValueError, "wrong anchor"):
            facts.validate_repository(self.root)
        self.write(self.document, rule())
        self.write(second, rule())
        with self.assertRaisesRegex(ValueError, "duplicate public IDs across topics"):
            facts.validate_repository(self.root)

    def test_uncataloged_topic_is_rejected(self):
        self.write("docs/gamefacts/EXTRA.md", rule("EXTRA-0001"))
        with self.assertRaisesRegex(ValueError, "uncataloged=.*EXTRA.md"):
            facts.validate_repository(self.root)

    def test_navigation_and_conventions_are_not_rule_topics(self):
        self.write("docs/gamefacts/README.md", "# Index\n")
        self.write("docs/gamefacts/CONVENTIONS.md", "# Conventions\n")
        facts.validate_repository(self.root)

    def test_missing_implementation_or_test_file_is_rejected(self):
        for path in ("internal/sample.go", "internal/sample_test.go"):
            with self.subTest(path=path):
                (self.root / path).unlink()
                with self.assertRaisesRegex(ValueError, "missing implementation/test locator"):
                    facts.validate_repository(self.root)
                self.write(path, "package sample\n")

    def test_locators_cannot_escape_or_be_directories(self):
        self.write(self.document, rule().replace("internal/sample.go", "../escape.go"))
        with self.assertRaisesRegex(ValueError, "escapes"):
            facts.validate_repository(self.root)
        self.write(self.document, rule())
        (self.root / "internal/sample.go").unlink()
        (self.root / "internal/sample.go").mkdir()
        with self.assertRaisesRegex(ValueError, "missing implementation/test locator"):
            facts.validate_repository(self.root)

    def test_reference_annotations_must_resolve(self):
        for reference, expected in (("RULE-9999", "unknown ref"), ("rule-0001", "malformed ref"),
                                    ("RULE-00012", "malformed ref"), ("", "malformed ref")):
            with self.subTest(reference=reference):
                self.write("internal/sample.go", "package sample\n// ref:" + reference + "\n")
                with self.assertRaisesRegex(ValueError, expected):
                    facts.validate_repository(self.root)

    def test_duplicate_catalog_keys_are_rejected(self):
        self.write("docs/gamefacts/catalog.json", '{"schema_version": 1, "schema_version": 1}')
        with self.assertRaisesRegex(ValueError, "duplicate JSON key"):
            facts.validate_repository(self.root)


if __name__ == "__main__":
    unittest.main()
