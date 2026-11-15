package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/kitew"
	"github.com/kite-plus/kite/internal/latest"
	"github.com/kite-plus/kite/internal/project"
)

func newVersionCmd() *cobra.Command {
	var check, short bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stampVCS()
			if short {
				printf(cmd, "%s\n", buildinfo.Version)
				return nil
			}
			info := map[string]any{
				"version":   buildinfo.Version,
				"commit":    buildinfo.Commit,
				"date":      buildinfo.Date,
				"go":        runtime.Version(),
				"platform":  runtime.GOOS + "/" + runtime.GOARCH,
				"themeAPI":  buildinfo.ThemeAPIVersion,
				"pluginABI": buildinfo.PluginABIVersion,
				"buildABI":  buildinfo.BuildABIVersion,
			}
			note := ""
			if check {
				note = newestNote(cmd.Context(), info)
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), info)
			}
			printf(cmd, "kite %s (%s, %s) %s %s\n", buildinfo.Version, buildinfo.Commit,
				buildinfo.Date, runtime.Version(), info["platform"])
			printf(cmd, "theme api %s, plugin abi %d\n", buildinfo.ThemeAPIVersion, buildinfo.PluginABIVersion)
			if check {
				printf(cmd, "%s\n", note)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&check, "check", "c", false,
		"ask GitHub whether a newer release is out")
	cmd.Flags().BoolVarP(&short, "short", "s", false,
		"print the version alone")
	return cmd
}

// stampVCS fills the commit and the date from the VCS information the Go
// toolchain embeds, for a binary linked without the flags make build passes.
// A release number cannot be recovered that way, so such a build still
// reports itself as dev.
func stampVCS() {
	if buildinfo.Commit != "none" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 12 {
				s.Value = s.Value[:12]
			}
			buildinfo.Commit = s.Value
		case "vcs.time":
			buildinfo.Date = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				buildinfo.Commit += "-dirty"
			}
		}
	}
}

// newestNote answers whether a newer release is out, keeping what it found in
// info for --json and returning the line a person reads.
//
// The answer is cached for a day, so a check run twice asks GitHub once. A
// build from source has no release to compare with, so the newest one is named
// rather than a verdict about it.
func newestNote(ctx context.Context, info map[string]any) string {
	if latest.Off() {
		info["check"] = "off"
		return "update checks are off (KITE_NO_UPDATE_CHECK)"
	}

	newest, err := latest.New().Check(ctx)
	if err != nil {
		info["check"] = "failed"
		return "could not ask GitHub for the newest release: " + err.Error()
	}
	info["newest"] = newest.Version
	info["newestURL"] = newest.URL

	running, released := kitew.Release(buildinfo.Version)
	switch {
	case !released:
		info["check"] = "source"
		return "this is a build from source; the newest release is " + newest.Version + " (" + newest.URL + ")"
	case latest.Newer(newest.Version, running):
		info["check"] = "newer"
		return "a newer release is out: " + newest.Version + " (" + newest.URL + ")"
	default:
		info["check"] = "current"
		return "this is the newest release"
	}
}

func newNewCmd() *cobra.Command {
	var draft bool
	cmd := &cobra.Command{
		Use:   "new <kind> <title>",
		Short: "Create a content item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			t, err := p.TypeOrError(args[0])
			if err != nil {
				return err
			}

			status := content.StatusPublished
			if draft {
				status = content.StatusDraft
			}
			now := time.Now().UTC().Truncate(time.Second)
			item := &content.Content{
				Kind:   t.Kind,
				Title:  args[1],
				Status: status,
				Body:   content.Body{Format: content.FormatMarkdown, Raw: "\n"},
			}
			if status == content.StatusPublished {
				item.PublishedAt = &now
			}

			res, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PutContent{Content: item}},
				Message: "new: " + item.Title,
			})
			if err != nil {
				return err
			}

			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"id":      item.ID,
					"kind":    item.Kind,
					"slug":    item.Slug,
					"locator": item.Locator,
					"written": res.Written,
				})
			}
			for _, w := range res.Written {
				printf(cmd, "%s\n", w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&draft, "draft", false, "create the item as a draft")
	return cmd
}

type doctorReport struct {
	Root       string   `json:"root"`
	Items      int      `json:"items"`
	MissingIDs []string `json:"missingIds,omitempty"`
	Problems   []string `json:"problems,omitempty"`
	// Packages are what kite.lock says that the installed themes and
	// plugins do not bear out.
	Packages []string `json:"packages,omitempty"`
	// Kite is what keeps the release kite.lock pins from being the one
	// every build runs.
	Kite  []string `json:"kite,omitempty"`
	Fixed []string `json:"fixed,omitempty"`
}

func newDoctorCmd() *cobra.Command {
	var fixIDs bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the project for problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			scan, err := p.Scanner().Scan()
			if err != nil {
				// A duplicate ID is reported, never repaired: only the author
				// knows which copy should keep the identity.
				return err
			}

			report := doctorReport{Root: p.Root, Items: len(scan.Entries), Packages: lockProblems(p.Root), Kite: pinProblems(p.Root)}
			for _, pr := range scan.Problems {
				report.Problems = append(report.Problems, pr.Error())
			}

			missing := scan.MissingIDs()
			for _, e := range missing {
				report.MissingIDs = append(report.MissingIDs, e.Path)
			}

			if fixIDs && len(missing) > 0 {
				var cs content.ChangeSet
				for _, e := range missing {
					e.Item.ID = content.NewID()
					cs.Add(content.PutContent{Content: e.Item, IfRevision: e.Item.Revision})
				}
				cs.Message = "doctor: assign content ids"
				res, err := p.Writer().Apply(cmd.Context(), cs)
				if err != nil {
					return err
				}
				report.Fixed = res.Written
				report.MissingIDs = nil
			}

			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			return printDoctor(cmd, report, fixIDs)
		},
	}
	cmd.Flags().BoolVar(&fixIDs, "fix-ids", false, "assign an id to every item that lacks one")
	return cmd
}

func printDoctor(cmd *cobra.Command, r doctorReport, fixed bool) error {
	printf(cmd, "project  %s\n", r.Root)
	printf(cmd, "items    %d\n", r.Items)

	if len(r.Problems) > 0 {
		printf(cmd, "\nunreadable files:\n")
		for _, p := range r.Problems {
			printf(cmd, "  %s\n", p)
		}
	}
	if len(r.Packages) > 0 {
		printf(cmd, "\nthemes and plugins:\n")
		for _, p := range r.Packages {
			printf(cmd, "  %s\n", p)
		}
	}
	if len(r.Kite) > 0 {
		printf(cmd, "\nKite release:\n")
		for _, p := range r.Kite {
			printf(cmd, "  %s\n", p)
		}
	}
	if len(r.Fixed) > 0 {
		printf(cmd, "\nassigned ids to %d file(s):\n", len(r.Fixed))
		for _, f := range r.Fixed {
			printf(cmd, "  %s\n", f)
		}
	}
	if len(r.MissingIDs) > 0 {
		printf(cmd, "\n%d file(s) have no id:\n", len(r.MissingIDs))
		for _, m := range r.MissingIDs {
			printf(cmd, "  %s\n", m)
		}
		printf(cmd, "\nrun 'kite doctor --fix-ids' to assign them\n")
	}
	if len(r.Problems) == 0 && len(r.MissingIDs) == 0 && len(r.Packages) == 0 && len(r.Kite) == 0 && !fixed {
		printf(cmd, "\nno problems found\n")
	}
	return nil
}

func openProject() (*project.Project, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	p, err := project.Open(wd)
	if errors.Is(err, project.ErrNotFound) {
		return nil, fmt.Errorf("%w\n\nrun 'kite init' to create one", err)
	}
	return p, err
}
