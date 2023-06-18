package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nao1215/imaging"
	"github.com/spf13/cobra"
)

func newBlurCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "blur",
		Short: "Blur the image according to sigma",
		Long: `Blur the image according to sigma.

The file extension specified in the --output parameter can be different from the input image's extension.`,
		Example: "   gina blur --sigma 2.0 input.jpg",
		RunE:    blur,
	}

	cmd.Flags().Float64P("sigma", "s", 0.0, "sigma parameter allows to control the strength of the blurring effect")
	cmd.Flags().StringP("output", "o", "output.jpg", "output filename (supported format: jpg, png, gif, tiff, bmp)")

	return &cmd
}

// resize have options for resize image.
type blurer struct {
	sigma  float64
	input  string
	output string
}

// newBlurer returns a new blurer. It returns an error if the required options are not set.
func newBlurer(cmd *cobra.Command, args []string) (*blurer, error) {
	s, err := cmd.Flags().GetFloat64("sigma")
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

	return &blurer{
		sigma:  s,
		input:  args[0],
		output: o,
	}, nil
}

func blur(cmd *cobra.Command, args []string) error {
	blurer, err := newBlurer(cmd, args)
	if err != nil {
		return err
	}
	return blurer.blur()
}

func (r *blurer) blur() error {
	src, err := imaging.Open(r.input)
	if err != nil {
		return err
	}

	dst := imaging.Blur(src, r.sigma)
	fmt.Fprintf(os.Stdout, "save image: %s\n", r.output)
	return imaging.Save(dst, r.output)
}
