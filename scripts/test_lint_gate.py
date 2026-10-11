import subprocess
import unittest

from lint_gate import run_inventory


class LintInventoryTests(unittest.TestCase):
    def test_complete_unique_inventory_survives_an_earlier_finding(self):
        inventory = [f"example/pkg{i}" for i in range(9)]
        calls = []

        def run(command, **options):
            calls.append(command)
            if command[:2] == ["go", "list"]:
                return subprocess.CompletedProcess(command, 0, "\n".join(inventory))
            self.assertFalse(options["check"])
            return subprocess.CompletedProcess(command, 1 if len(calls) == 2 else 0)

        self.assertEqual(run_inventory("linter", ["./..."], runner=run), 1)
        checked = [item for call in calls[1:-1] for item in call if item.startswith("example/")]
        self.assertEqual(checked, inventory)
        self.assertIn("--enable-only=dupl", calls[-1])
        self.assertEqual(calls[-1][-len(inventory):], inventory)
        self.assertTrue(all("--new=false" not in call for call in calls[1:]))

    def test_empty_or_duplicate_inventory_fails_before_lint(self):
        for output in ("", "example/a\nexample/a"):
            calls = []

            def run(command, **options):
                calls.append(command)
                return subprocess.CompletedProcess(command, 0, output)

            with self.assertRaises(ValueError):
                run_inventory("linter", ["./..."], runner=run)
            self.assertEqual(len(calls), 1)

    def test_unfiltered_mode_preserves_uncapped_policy(self):
        calls = []

        def run(command, **options):
            calls.append(command)
            return subprocess.CompletedProcess(command, 0, "example/a")

        self.assertEqual(run_inventory("linter", ["./..."], all_findings=True, runner=run), 0)
        self.assertIn("--new-from-rev=", calls[1])
        self.assertIn("--max-issues-per-linter=0", calls[1])
        self.assertIn("--max-same-issues=0", calls[1])
        self.assertIn("--uniq-by-line=false", calls[1])


if __name__ == "__main__":
    unittest.main()
