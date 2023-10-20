package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-spectest/imaging"
	"github.com/spf13/cobra"
)

func newContrastCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "contrast",
		Short:   "Adjust the contrast of an image",
		Example: "   gina contrast -p 20 -o output.png input.jpg",
		RunE:    adjustContrast,
	}

	cmd.Flags().Float64P("percentage", "p", 0, "percentage = 0 gives the original image. range (-100, 100)")
	cmd.Flags().StringP("output", "o", "output.jpg", "output filename (supported format: jpg, png, gif, tiff, bmp)")

	return &cmd
}

// contraster have options for adjusting cotrast of image.
type contraster struct {
	percentage float64
	input      string
	output     string
}

// newContraster returns a new contraster. It returns an error if the required options are not set.
func newContraster(cmd *cobra.Command, args []string) (*contraster, error) {
	p, err := cmd.Flags().GetFloat64("percentage")
	if err != nil {
		return nil, err
	}

	o, err := cmd.Flags().GetString("output")
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, errors.New("no argument: input image file path is required")
	}

	return &contraster{
		percentage: p,
		input:      args[0],
		output:     o,
	}, nil
}

func adjustContrast(cmd *cobra.Command, args []string) error {
	contraster, err := newContraster(cmd, args)
	if err != nil {
		return err
	}
	return contraster.contrast()
}

func (c *contraster) contrast() error {
	src, err := imaging.Open(c.input)
	if err != nil {
		return err
	}

	dst := imaging.AdjustContrast(src, float64(c.percentage))
	fmt.Fprintf(os.Stdout, "save image: %s\n", c.output)
	return imaging.Save(dst, c.output)
}
