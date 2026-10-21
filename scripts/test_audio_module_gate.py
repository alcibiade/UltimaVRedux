"""Regression checks for published SDK pinning and the game-only substitution."""

# Descriptive unittest names document these small assertion-only methods.
# pylint: disable=missing-function-docstring

import copy
import unittest

from audio_module_gate import (
    MODULE,
    TOOLS,
    ROOT,
    ModuleMetadata,
    ModuleVersion,
    SelectedModule,
    sdk_version,
    verify_bridge_replacements,
    verify_selected_sdk,
)


class AudioModuleGateTests(unittest.TestCase):
    """Reject every substitution outside the reviewed game bridge."""

    def metadata(self) -> ModuleMetadata:
        return {
            "Replace": [
                {
                    "Old": {"Path": "github.com/bradhannah/UltimaVRedux"},
                    "New": {"Path": str(ROOT)},
                },
            ]
        }

    def test_only_known_test_replacements_are_accepted(self) -> None:
        verify_bridge_replacements(self.metadata())
        for change in ("extra", "version", "outside", "duplicate", "missing"):
            with self.subTest(change=change):
                metadata = copy.deepcopy(self.metadata())
                if change == "extra":
                    metadata["Replace"].append(
                        {
                            "Old": {"Path": "example.com/other"},
                            "New": {"Path": "../other"},
                        }
                    )
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

    def test_sdk_cannot_be_substituted_or_silently_upgraded(self) -> None:
        version = "v0.0.0-20261018221138-9e7e3ca45b46"
        selected: SelectedModule = {"Path": MODULE, "Version": version}
        verify_selected_sdk(selected, version)
        for change in (
            {"Replace": {"Path": "../sdk"}},
            {"Version": "v0.1.0"},
            {"Path": "example.com/other"},
        ):
            with self.subTest(change=change), self.assertRaises(ValueError):
                verify_selected_sdk({**selected, **change}, version)
        metadata = self.metadata()
        metadata["Replace"].append(
            {"Old": {"Path": MODULE}, "New": {"Path": "../library"}}
        )
        with self.assertRaises(ValueError):
            verify_bridge_replacements(metadata)

    def test_fixture_requires_a_concrete_sdk_pin(self) -> None:
        pinned: ModuleVersion = {"Path": MODULE, "Version": "v0.1.0"}
        self.assertEqual(sdk_version({"Require": [pinned]}), "v0.1.0")
        for requirements in (
            [],
            [pinned, pinned],
            [{"Path": MODULE, "Version": "v0.0.0"}],
        ):
            with self.subTest(requirements=requirements), self.assertRaises(ValueError):
                sdk_version({"Require": requirements})

    def test_tools_pin_and_actual_selection_are_independent_of_runtime_pin(
        self,
    ) -> None:
        metadata: ModuleMetadata = {
            "Require": [
                {"Path": MODULE, "Version": "v0.1.0"},
                {"Path": TOOLS, "Version": "v0.2.0"},
            ]
        }
        self.assertEqual(sdk_version(metadata, TOOLS), "v0.2.0")
        verify_selected_sdk({"Path": TOOLS, "Version": "v0.2.0"}, "v0.2.0", TOOLS)
        with self.assertRaises(ValueError):
            verify_selected_sdk({"Path": TOOLS, "Version": "v0.3.0"}, "v0.2.0", TOOLS)
