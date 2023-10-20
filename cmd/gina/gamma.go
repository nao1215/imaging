package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-spectest/imaging"
	"github.com/spf13/cobra"
)

func newGammaCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "gamma",
		Short:   "Adjust the gamma correction of an image",
		Example: "   gina gamma --gamma 0.75 -o output.png input.jpg",
		RunE:    gamma,
	}

	cmd.Flags().Float64P("gamma", "g", 0, "gamma less than 1.0 darkens the image and gamma greater than 1.0 lightens it")
	cmd.Flags().StringP("output", "o", "output.jpg", "output filename (supported format: jpg, png, gif, tiff, bmp)")

	return &cmd
}

// gammer have options for adjusting the gamma correction of an image
type gammer struct {
	gamma  float64
	input  string
	output string
}

// newGammer returns a new gammer. It returns an error if the required options are not set.
func newGammer(cmd *cobra.Command, args []string) (*gammer, error) {
	g, err := cmd.Flags().GetFloat64("gamma")
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

	return &gammer{
		gamma:  g,
		input:  args[0],
		output: o,
	}, nil
}

func gamma(cmd *cobra.Command, args []string) error {
	gammer, err := newGammer(cmd, args)
	if err != nil {
		return err
	}
	return gammer.adjustGammer()
}

func (g *gammer) adjustGammer() error {
	src, err := imaging.Open(g.input)
	if err != nil {
		return err
	}

	dst := imaging.AdjustGamma(src, g.gamma)
	fmt.Fprintf(os.Stdout, "save image: %s\n", g.output)
	return imaging.Save(dst, g.output)
}
