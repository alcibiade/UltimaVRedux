import contextlib
import io
import subprocess
import unittest

from lint_gate import host_concurrency, run_inventory


class LintInventoryTests(unittest.TestCase):
    def test_host_checks_requested_inventory_once_with_automatic_cpus(self):
        inventory = [f"example/pkg{i}" for i in range(9)]
        calls = []

        def run(command, **options):
            calls.append(command)
            if command[:2] == ["go", "list"]:
                self.assertEqual(command[4:], ["./internal/...", "./scripts/..."])
                return subprocess.CompletedProcess(command, 0, "\n".join(inventory))
            self.assertFalse(options["check"])
            return subprocess.CompletedProcess(command, 37)

        self.assertEqual(run_inventory("linter", ["./internal/...", "./scripts/..."],
                                       runner=run, environment={}), 37)
        self.assertEqual(len(calls), 2)
        self.assertEqual(calls[1][1:], ["run", *inventory])
        self.assertNotIn("--enable-only=dupl", calls[1])
        self.assertNotIn("--disable=dupl", calls[1])

    def test_host_preserves_explicit_resources_and_reports_only_resources(self):
        calls = []

        def run(command, **options):
            calls.append(command)
            return subprocess.CompletedProcess(command, 0, "example/a")

        environment = {"GOMAXPROCS": "7", "GOMEMLIMIT": "768MiB", "GOGC": "125", "SECRET": "hidden"}
        with contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(run_inventory("linter", ["./..."], runner=run, environment=environment), 0)
        self.assertEqual(calls[1][1:], ["run", "--concurrency=7", "example/a"])
        self.assertIn("concurrency=7", output.getvalue())
        self.assertIn("GOMEMLIMIT='768MiB' GOGC='125'", output.getvalue())
        self.assertNotIn("hidden", output.getvalue())
        self.assertEqual(environment["GOMAXPROCS"], "7")

    def test_host_cpu_defaults_follow_go_environment_semantics(self):
        for value in ("", "0", "-1", "invalid"):
            self.assertEqual(host_concurrency({"GOMAXPROCS": value}), [])
        self.assertEqual(host_concurrency({"GOMAXPROCS": "5"}), ["--concurrency=5"])

    def test_complete_unique_inventory_survives_an_earlier_finding(self):
        inventory = [f"example/pkg{i}" for i in range(9)]
        calls = []

        def run(command, **options):
            calls.append(command)
            if command[:2] == ["go", "list"]:
                return subprocess.CompletedProcess(command, 0, "\n".join(inventory))
            self.assertFalse(options["check"])
            return subprocess.CompletedProcess(command, 1 if len(calls) == 2 else 0)

        self.assertEqual(run_inventory("linter", ["./..."], runner=run, profile="constrained"), 1)
        checked = [item for call in calls[1:-1] for item in call if item.startswith("example/")]
        self.assertEqual(checked, inventory)
        self.assertIn("--enable-only=dupl", calls[-1])
        self.assertEqual(calls[-1][-len(inventory):], inventory)
        self.assertTrue(all("--new=false" not in call for call in calls[1:]))
        self.assertTrue(all("--disable=dupl" in call for call in calls[1:-1]))
        self.assertTrue(all("--concurrency=1" in call for call in calls[1:]))
        self.assertNotIn("--disable=dupl", calls[-1])

    def test_throughput_retains_serial_fallback_and_global_duplicate_failure(self):
        calls = []

        def run(command, **options):
            calls.append(command)
            return subprocess.CompletedProcess(command, 1 if "--enable-only=dupl" in command else 0,
                                               "example/a\nexample/b")

        self.assertEqual(run_inventory("linter", ["./..."], runner=run, profile="throughput"), 1)
        self.assertEqual(len(calls), 3)
        self.assertEqual(calls[-1][-2:], ["example/a", "example/b"])
        self.assertIn("--concurrency=1", calls[1])

    def test_cancelled_linter_does_not_start_another_batch(self):
        calls = []

        def run(command, **options):
            calls.append(command)
            return subprocess.CompletedProcess(command, 0 if len(calls) == 1 else -15,
                                               "\n".join(f"example/pkg{i}" for i in range(9)))

        self.assertEqual(run_inventory("linter", ["./..."], runner=run, profile="constrained"), 143)
        self.assertEqual(len(calls), 2)

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
        for profile in ("host", "constrained", "throughput"):
            with self.subTest(profile=profile):
                calls = []

                def run(command, **options):
                    calls.append(command)
                    return subprocess.CompletedProcess(command, 0, "example/a")

                self.assertEqual(run_inventory("linter", ["./..."], all_findings=True, runner=run,
                                               profile=profile), 0)
                for command in calls[1:]:
                    for flag in ("--new=false", "--new-from-rev=", "--new-from-patch=", "--new-from-merge-base=",
                                 "--max-issues-per-linter=0", "--max-same-issues=0", "--uniq-by-line=false"):
                        self.assertIn(flag, command)

    def test_unknown_profile_fails_before_commands(self):
        def run(*args, **kwargs):
            self.fail("invalid profile cannot start a command")

        with self.assertRaisesRegex(ValueError, "Unknown lint resource profile"):
            run_inventory("linter", ["./..."], runner=run, profile="invalid")


if __name__ == "__main__":
    unittest.main()
