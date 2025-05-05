# Dynamic Interval Feature Implementation Plan

## Overview

Currently, chaoskube kills a pod and then waits for a fixed interval before killing another pod. The goal is to implement a dynamic interval feature that adjusts the interval based on the number of pods in the cluster. A cluster with more pods should have a shorter interval between pod terminations, while a cluster with fewer pods should have a longer interval.

## Current Implementation

In the current implementation:
1. The user specifies a fixed interval using the `--interval` flag (default: 10m)
2. Chaoskube terminates a victim pod (or multiple pods if `--max-kill` > 1)
3. Chaoskube sleeps for the specified interval
4. The process repeats

## Proposed Solution

We will implement a dynamic interval calculation that adjusts the interval based on the number of pods in the cluster:

1. Add new command-line flags:
   - `--dynamic-interval`: Boolean flag to enable/disable dynamic interval (default: false)
   - `--dynamic-interval-factor`: Float value to adjust the impact of pod count on interval (default: 1.0)

2. When dynamic interval is enabled:
   - Use the existing `--interval` flag as the base interval
   - Calculate a new interval based on the formula: `newInterval = baseInterval * (referencePodCount / actualPodCount)^factor`
   - Use 10 pods as a reference point:
     - With 10 pods: interval = baseInterval
     - With 20 pods: interval = baseInterval / (2^factor)
     - With 5 pods: interval = baseInterval * (2^factor)

3. The factor parameter allows users to adjust how dramatically the interval changes:
   - factor = 1.0: Linear relationship (default)
   - factor > 1.0: More dramatic changes
   - factor < 1.0: Less dramatic changes

## Implementation Steps

1. **Update Chaoskube Struct**:
   - Add fields for dynamic interval configuration:
     - `DynamicInterval` (bool)
     - `DynamicIntervalFactor` (float64)
     - `BaseInterval` (time.Duration)

2. **Add Command-Line Flags**:
   - Add `--dynamic-interval` flag (boolean)
   - Add `--dynamic-interval-factor` flag (float64)
   - Use the existing `--interval` flag as the base interval

3. **Implement Dynamic Interval Calculation**:
   - Create a `CalculateDynamicInterval` method that:
     - Returns the base interval if dynamic interval is disabled
     - Calculates a new interval based on the pod count and factor
     - Uses the formula: `newInterval = baseInterval * (10 / podCount)^factor`
     - Handles edge cases (e.g., no pods)

4. **Modify the Run Method**:
   - Update the Run method to use the dynamic interval when enabled
   - When dynamic interval is enabled, calculate the interval before each termination
   - Use `time.After()` with the calculated interval instead of the ticker

5. **Add Metrics**:
   - Add a metric to track the current interval
   - This will help users monitor how the interval changes over time

6. **Update Tests**:
   - Add tests for the dynamic interval calculation
   - Test different pod counts and factor values
   - Test edge cases

## Formula Explanation

The formula `newInterval = baseInterval * (referencePodCount / actualPodCount)^factor` works as follows:

- When `actualPodCount` equals `referencePodCount` (10 pods), the ratio is 1, so the interval remains the same as the base interval.
- When `actualPodCount` is greater than `referencePodCount`, the ratio is less than 1, so the interval decreases.
- When `actualPodCount` is less than `referencePodCount`, the ratio is greater than 1, so the interval increases.
- The `factor` parameter allows adjusting the curve of this relationship.

This creates a balanced approach where:
- Clusters with many pods have more frequent terminations (shorter intervals)
- Clusters with few pods have less frequent terminations (longer intervals)
- The relationship is intuitive and predictable

## Benefits

1. **Adaptive Chaos**: The chaos level automatically adjusts to the cluster size
2. **Balanced Impact**: Maintains a consistent level of chaos regardless of cluster size
3. **Configurable**: Users can adjust the factor to control how dramatically the interval changes
4. **Backward Compatible**: Users can disable the feature to maintain the current behavior

## Conclusion

This implementation provides a flexible and intuitive way to adjust the interval between pod terminations based on the number of pods in the cluster. It maintains backward compatibility while adding a powerful new feature that makes chaoskube more adaptive to different cluster sizes.
