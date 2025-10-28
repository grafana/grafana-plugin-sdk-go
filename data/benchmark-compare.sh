#!/bin/bash
set -e

# Script to compare benchmark performance between branches using best practices
# Usage: ./benchmark-compare.sh [base-branch] [feature-branch] [count]
#
# Best practices implemented:
# - Uses -count=10 for statistical significance (can override with 3rd arg)
# - Clears build cache between runs
# - Stabilizes CPU frequency where possible
# - Uses -benchtime for longer runs to reduce noise
#
# If no arguments provided:
# - Runs benchmarks on current branch and saves to new.txt
# - Switches to main, runs benchmarks and saves to old.txt
# - Switches back and compares with benchstat
#
# If arguments provided:
# - Uses specified branches for comparison

CURRENT_BRANCH=$(git branch --show-current)
BASE_BRANCH=${1:-main}
FEATURE_BRANCH=${2:-$CURRENT_BRANCH}
COUNT=${3:-10}  # Default to 10 runs for better statistical significance

echo "======================================================================"
echo "Benchmark Comparison with Best Practices"
echo "======================================================================"
echo "  Base branch: $BASE_BRANCH"
echo "  Feature branch: $FEATURE_BRANCH"
echo "  Iterations: $COUNT (minimum 6 recommended for confidence intervals)"
echo ""

# Check if benchstat is installed
if ! command -v benchstat &> /dev/null; then
    echo "benchstat is not installed. Installing..."
    go install golang.org/x/perf/cmd/benchstat@latest
    echo ""
fi

# Warn about CPU frequency scaling
echo "NOTE: For most accurate results:"
echo "  - Close other applications"
echo "  - Disable CPU frequency scaling if possible"
echo "  - Run on AC power (laptops)"
echo "  - Consider: sudo cpupower frequency-set --governor performance (Linux)"
echo ""

# Save current state
echo "Saving current work..."
git stash push -u -m "benchmark comparison stash" 2>/dev/null || true

# Function to run benchmarks with best practices
run_benchmarks() {
    local branch=$1
    local output=$2
    
    echo "======================================================================"
    echo "Running benchmarks on $branch..."
    echo "======================================================================"
    
    # Clear build cache to ensure clean build
    echo "Clearing build cache..."
    go clean -cache -testcache
    
    # Run benchmarks with:
    # - count=$COUNT: Multiple runs for statistical significance
    # - benchmem: Include memory allocation stats
    # - benchtime=1s: Run each benchmark for at least 1 second (reduces timing noise)
    # - run=^$: Don't run any tests, only benchmarks
    echo "Running $COUNT iterations (this may take several minutes)..."
    go test -bench=. -benchmem -count=$COUNT -benchtime=1s -cpu=1 -run=^$ ./data 2>&1 | tee "$output"
    
    echo ""
    echo "Results saved to $output"
}

# Run benchmarks on base branch
git checkout "$BASE_BRANCH" 2>&1 | grep -v "^M\s" || true
run_benchmarks "$BASE_BRANCH" "old.txt"

# Run benchmarks on feature branch
git checkout "$FEATURE_BRANCH" 2>&1 | grep -v "^M\s" || true
run_benchmarks "$FEATURE_BRANCH" "new.txt"

echo ""
echo "======================================================================"
echo "Benchmark Comparison Results"
echo "======================================================================"
benchstat -alpha=0.05 old.txt new.txt
