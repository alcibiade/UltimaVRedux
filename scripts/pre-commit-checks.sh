#!/bin/bash

# Pre-commit checks for repository-wide architecture and quality invariants
# Run this script before committing to catch common violations

set -e
set -o pipefail

POLICY_ONLY=false
if [ "$#" -gt 1 ] || { [ "$#" -eq 1 ] && [ "$1" != "--policy-only" ]; }; then
    echo "Usage: $0 [--policy-only]" >&2
    exit 2
fi
if [ "${1:-}" = "--policy-only" ]; then
    POLICY_ONLY=true
fi

echo "🔍 Running pre-commit validation checks..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track overall success
OVERALL_SUCCESS=true

# Track timing
START_TIME=$(date +%s)

#############################################
# Check 1: No time.Now() in core logic
#############################################
echo -e "\n📅 Checking for time.Now() usage in core logic..."

CORE_PACKAGES=(
    "internal/game_state"
    "internal/ai"
    "internal/sprites"
    "internal/map_state"
    "internal/party_state"
)

TIME_NOW_VIOLATIONS=0
for package in "${CORE_PACKAGES[@]}"; do
    if [ ! -d "$package" ]; then
        echo -e "${YELLOW}⚠️  Warning: Package directory $package not found${NC}"
        continue
    fi

    violations=$(grep -rn "time\.Now()" "$package" 2>/dev/null | grep -v "_test.go" | grep -v "// ALLOWED:" | grep -v "// Use deterministic RNG instead of time.Now()" || true)
    if [ -n "$violations" ]; then
        echo -e "${RED}❌ Found time.Now() usage in $package:${NC}"
        while IFS= read -r violation; do
            file_line=$(echo "$violation" | cut -d: -f1,2)
            echo -e "  ${RED}$file_line${NC}"
            # Show 2 lines of context
            file=$(echo "$violation" | cut -d: -f1)
            line_num=$(echo "$violation" | cut -d: -f2)
            sed -n "$((line_num-1)),$((line_num+1))p" "$file" 2>/dev/null | sed "s/^/    /" || true
        done <<< "$violations"
        TIME_NOW_VIOLATIONS=1
    fi
done

if [ $TIME_NOW_VIOLATIONS -eq 0 ]; then
    echo -e "${GREEN}✅ No time.Now() violations in core logic${NC}"
else
    echo -e "${RED}❌ time.Now() usage found in core logic packages${NC}"
    echo -e "${YELLOW}💡 Use GameState.DateTime or central GameClock instead${NC}"
    OVERALL_SUCCESS=false
fi

#############################################
# Check 2: log.Fatal has comments
#############################################
echo -e "\n🚨 Checking log.Fatal comments..."

FATAL_VIOLATIONS=0
fatal_calls=$(grep -rn "log\.Fatal" --include="*.go" . 2>/dev/null | \
    grep -v "_test.go" | \
    grep -v "//.*log\.Fatal" || true)

if [ -n "$fatal_calls" ]; then
    while IFS= read -r line; do
        file_line=$(echo "$line" | cut -d: -f1,2)

        # Get the line before the log.Fatal call to check for comment
        file=$(echo "$line" | cut -d: -f1)
        line_num=$(echo "$line" | cut -d: -f2)
        prev_line_num=$((line_num - 1))

        if [ $prev_line_num -gt 0 ]; then
            prev_line=$(sed -n "${prev_line_num}p" "$file" 2>/dev/null || echo "")
            # Check if previous line has TODO or explanatory comment
            if echo "$prev_line" | grep -q "// TODO\|// .*:" && ! echo "$prev_line" | grep -q "^[[:space:]]*$"; then
                # Has appropriate comment
                continue
            fi
        fi

        echo -e "${RED}❌ log.Fatal without appropriate comment: $file_line${NC}"
        FATAL_VIOLATIONS=1
    done <<< "$fatal_calls"
fi

if [ $FATAL_VIOLATIONS -eq 0 ]; then
    echo -e "${GREEN}✅ All log.Fatal calls have appropriate comments${NC}"
else
    echo -e "${RED}❌ Found log.Fatal calls without comments${NC}"
    echo -e "${YELLOW}💡 Add explanatory comment or '// TODO: soften to recoverable error'${NC}"
    OVERALL_SUCCESS=false
fi

#############################################
# Check 3: Import organization
#############################################
echo -e "\n📦 Checking import organization..."

# Check if goimports would make changes
IMPORT_VIOLATIONS=0
if [ "$POLICY_ONLY" = false ]; then
    if ! import_output=$(make format-check 2>&1); then
        echo -e "${RED}❌ goimports check failed:${NC}"
        echo "$import_output"
        IMPORT_VIOLATIONS=1
        OVERALL_SUCCESS=false
    fi

    if [ $IMPORT_VIOLATIONS -eq 0 ]; then
        echo -e "${GREEN}✅ Import organization correct${NC}"
    fi
fi

#############################################
# Check 4: Package boundary violations
#############################################
echo -e "\n🏗️ Checking package boundary violations..."

BOUNDARY_VIOLATIONS=0
if ! boundary_violations=$(make core-dependencies-check 2>&1); then
    echo -e "${RED}❌ Core production/test dependency boundary violation:${NC}"
    echo "$boundary_violations"
    echo -e "${YELLOW}💡 Keep runtime display/input/rendering behind presentation adapters${NC}"
    BOUNDARY_VIOLATIONS=1
    OVERALL_SUCCESS=false
else
    echo -e "${GREEN}✅ No package boundary violations${NC}"
fi

#############################################
# Check 5: Unnecessary import aliases
#############################################
echo -e "\n🔗 Checking for unnecessary import aliases..."

ALIAS_VIOLATIONS=0
bad_aliases=$(grep -rEn '^[[:space:]]*(references2|mainscreen2)[[:space:]]+"' --include="*.go" . || true)

if [ -n "$bad_aliases" ]; then
    echo -e "${RED}❌ Found unnecessary import aliases:${NC}"
    echo "$bad_aliases"
    echo -e "${YELLOW}💡 Remove unnecessary aliases like references2 and mainscreen2${NC}"
    ALIAS_VIOLATIONS=1
    OVERALL_SUCCESS=false
else
    echo -e "${GREEN}✅ No unnecessary import aliases found${NC}"
fi

#############################################
# Check 6: No direct rand usage in core logic
#############################################
echo -e "\n🎲 Checking for direct rand package usage in core logic..."

RAND_VIOLATIONS=0
for package in "${CORE_PACKAGES[@]}"; do
    if [ ! -d "$package" ]; then
        continue
    fi

    violations=$(grep -rn "rand\." "$package" 2>/dev/null | \
        grep -v "_test.go" | \
        grep -v "import \"math/rand\"" | \
        grep -v "import \"math/rand/v2\"" | \
        grep -v "// ALLOWED:" | \
        grep -v "Centralized deterministic" | \
        grep -v "rng.*\*rand\.Rand" | \
        grep -v "rand\.New\|rand\.NewSource" | \
        grep -v "gs\..*[Rr]and\|GameState.*[Rr]and\|\.rng\." || true)
    if [ -n "$violations" ]; then
        echo -e "${RED}❌ Found direct rand usage in $package:${NC}"
        while IFS= read -r violation; do
            file_line=$(echo "$violation" | cut -d: -f1,2)
            echo -e "  ${RED}$file_line${NC}"
            # Show context
            file=$(echo "$violation" | cut -d: -f1)
            line_num=$(echo "$violation" | cut -d: -f2)
            sed -n "$((line_num-1)),$((line_num+1))p" "$file" 2>/dev/null | sed "s/^/    /" || true
        done <<< "$violations"
        RAND_VIOLATIONS=1
    fi
done

if [ $RAND_VIOLATIONS -eq 0 ]; then
    echo -e "${GREEN}✅ No direct rand usage in core logic${NC}"
else
    echo -e "${RED}❌ Direct rand usage found${NC}"
    echo -e "${YELLOW}💡 Use gs.RollDice(), gs.OneInXOdds(), or gs.RandomIntInRange() instead${NC}"
    OVERALL_SUCCESS=false
fi

#############################################
# Check 7: Centralized logging usage
#############################################
echo -e "\n📝 Checking for proper logging usage..."

LOGGING_VIOLATIONS=0
bad_logging=$(grep -rn "fmt\.Printf\|log\.Printf" --include="*.go" internal/ cmd/ 2>/dev/null | \
    grep -v "_test.go" | \
    grep -v "// ALLOWED:" | \
    grep -v "log\.Fatal" | \
    grep -v "log\.Fatalf" | \
    grep -v "internal/logging/logger.go" | \
    grep -v "cmd/ultimav/main.go" || true)

if [ -n "$bad_logging" ]; then
    echo -e "${RED}❌ Found direct console output instead of centralized logging:${NC}"
    while IFS= read -r violation; do
        file_line=$(echo "$violation" | cut -d: -f1,2)
        echo -e "  ${RED}$file_line${NC}"
    done <<< "$bad_logging"
    echo -e "${YELLOW}💡 Use logging.Debug/Info/Warn/Error with categories instead${NC}"
    echo -e "${YELLOW}   Example: logging.Debug(logging.CategoryUI, \"message: %v\", value)${NC}"
    LOGGING_VIOLATIONS=1
    OVERALL_SUCCESS=false
else
    echo -e "${GREEN}✅ Proper logging usage${NC}"
fi

#############################################
# Check 8: No deprecated DrawText usage
#############################################
echo -e "\n🔤 Checking for deprecated DrawText usage..."

DRAWTEXT_VIOLATIONS=0
drawtext_calls=$(grep -rn "\.DrawText\|\.DrawTextCenter\|\.DrawTextRightToLeft" cmd/ internal/ --include="*.go" 2>/dev/null | \
    grep -v "_test.go" | \
    grep -v "output_rendering.go" || true)

if [ -n "$drawtext_calls" ]; then
    echo -e "${RED}❌ Found deprecated DrawText usage (use Output pattern instead):${NC}"
    while IFS= read -r violation; do
        file_line=$(echo "$violation" | cut -d: -f1,2)
        echo -e "  ${RED}$file_line${NC}"
    done <<< "$drawtext_calls"
    echo -e "${YELLOW}💡 See docs/ui/ for text rendering contracts${NC}"
    echo -e "${YELLOW}   Use: output.DrawPanelContent() or output.DrawContinuousOutputTexOnXy()${NC}"
    DRAWTEXT_VIOLATIONS=1
    OVERALL_SUCCESS=false
else
    echo -e "${GREEN}✅ No deprecated DrawText usage${NC}"
fi

#############################################
# Check 9: No player-game pointer input
#############################################
echo -e "\nChecking for player-game mouse, wheel, or touch input..."

POINTER_INPUT_VIOLATIONS=0
if ! pointer_input_calls=$(go run ./scripts/check-player-pointer-input 2>&1); then
    echo -e "${RED}❌ Found pointer input in the keyboard-only player game:${NC}"
    while IFS= read -r violation; do
        echo -e "  ${RED}$violation${NC}"
    done <<< "$pointer_input_calls"
    echo -e "${YELLOW}💡 Player-facing Go UI is keyboard-only; browser Debug Workbench sources are outside this check${NC}"
    POINTER_INPUT_VIOLATIONS=1
    OVERALL_SUCCESS=false
else
    echo -e "${GREEN}✅ No player-game pointer input found${NC}"
fi

#############################################
# Check 10: Basic Go checks
#############################################
if [ "$POLICY_ONLY" = false ]; then
    echo -e "\n🔧 Running basic Go checks..."

    echo "Running go vet..."
    if ! go vet ./...; then
        echo -e "${RED}❌ go vet failed${NC}"
        OVERALL_SUCCESS=false
    else
        echo -e "${GREEN}✅ go vet passed${NC}"
    fi

    echo "Running go build..."
    if ! go build ./...; then
        echo -e "${RED}❌ go build failed${NC}"
        OVERALL_SUCCESS=false
    else
        echo -e "${GREEN}✅ go build passed${NC}"
    fi

    #############################################
    # Check 11: Run unit tests
    #############################################
    echo -e "\n🧪 Running unit tests (non-integration)..."

    if ! go test -short ./... -timeout 90s; then
        echo -e "${RED}❌ Unit tests failed${NC}"
        OVERALL_SUCCESS=false
    else
        echo -e "${GREEN}✅ Unit tests passed${NC}"
    fi
fi

#############################################
# Summary
#############################################
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo -e "\n📋 Pre-commit Check Summary:"
echo "============================================"
echo "Duration: ${DURATION}s"
echo ""

if [ "$OVERALL_SUCCESS" = true ]; then
    echo -e "${GREEN}🎉 All checks passed! Ready to commit.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some checks failed. Please fix the issues above before committing.${NC}"
    echo -e "\n🔧 Quick fixes:"
    [ $IMPORT_VIOLATIONS -eq 1 ] && echo "  • Run: make format"
    [ $FATAL_VIOLATIONS -eq 1 ] && echo "  • Add comments to log.Fatal calls (see docs/ERROR_HANDLING.md)"
    [ $TIME_NOW_VIOLATIONS -eq 1 ] && echo "  • Replace time.Now() with gs.DateTime or gs.ElapsedMs"
    [ $BOUNDARY_VIOLATIONS -eq 1 ] && echo "  • Keep runtime display/input/rendering behind presentation adapters"
    [ $ALIAS_VIOLATIONS -eq 1 ] && echo "  • Remove aliases: references2→references, mainscreen2→mainscreen"
    [ $RAND_VIOLATIONS -eq 1 ] && echo "  • Use gs.RollDice(), gs.OneInXOdds(), or gs.RandomIntInRange()"
    [ $LOGGING_VIOLATIONS -eq 1 ] && echo "  • Use: logging.Debug/Info/Warn/Error(logging.Category*, ...)"
    [ $DRAWTEXT_VIOLATIONS -eq 1 ] && echo "  • Use: output.DrawPanelContent() instead of DrawText methods"
    [ $POINTER_INPUT_VIOLATIONS -eq 1 ] && echo "  • Remove mouse, wheel, and touch APIs from player-facing Go UI"
    echo -e "\n📚 See AGENTS.md and docs/TESTING.md for policy and verification guidance"
    exit 1
fi
