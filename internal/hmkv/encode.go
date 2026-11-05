package hmkv

import (
	"context"
	"fmt"
	"os"
)

// encodeTitle encodes a single title using HandBrakeCLI, updating progress on the tracker.
// On success it populates params.EncodedFileSizeBytes with the output file size.
func encodeTitle(
	ctx context.Context,
	hb *HandBrakeCLI,
	tracker *progressTracker,
	params *EncodingParams,
) error {
	tracker.applyChange(params.TitleIndex, params.DiscId, func(status *titleStatus) {
		status.Encoding = InProgress
		status.EncodingProgress = 0
	})

	// Make sure the input file exists
	if _, err := os.Stat(params.MKVOutputPath); os.IsNotExist(err) {
		return fmt.Errorf("encoding input file %s does not exist", params.MKVOutputPath)
	}

	hbProgressUpdate := func(percent int) {
		tracker.applyChange(params.TitleIndex, params.DiscId, func(status *titleStatus) {
			status.EncodingProgress = percent
		})
	}

	encErr := hb.encode(ctx, params, hbProgressUpdate)
	if encErr != nil {
		return encErr
	}

	// Update progress for encoding completion
	tracker.applyChange(params.TitleIndex, params.DiscId, func(status *titleStatus) {
		status.Encoding = Complete
		status.EncodingProgress = 100
	})
	tracker.forceRefresh()

	if stat, err := os.Stat(params.HandBrakeOutputPath); err == nil {
		params.EncodedFileSizeBytes = stat.Size()
	}

	return nil
}
