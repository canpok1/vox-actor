package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

// ErrUsage はCLIの引数エラーを示すエラー。
// 呼び出し元はこのエラーを検出して終了コード2で終了すべき。
var ErrUsage = errors.New("usage error")

// runAudioProbe は probe が nil でなく dryRun でない場合に probe を実行する。
// 失敗時は "audio device unavailable: ..." エラーを返す。
func runAudioProbe(probe func() error, dryRun bool) error {
	if dryRun || probe == nil {
		return nil
	}
	if err := probe(); err != nil {
		return fmt.Errorf("audio device unavailable: %w", err)
	}
	return nil
}

// Deps はルートコマンドの依存を保持する。
type Deps struct {
	Act          *ActDeps
	Watch        *WatchDeps
	Say          *SayDeps
	ScriptAppend *ScriptAppendDeps
	ScriptWrite  *ScriptWriteDeps
	AudioCheck   *AudioCheckDeps
	ViewerCheck  *ViewerCheckDeps
	Viewer       *ViewerDeps
	Assets       *AssetsDownloadDeps
	Speakers     *SpeakersDeps
}

func makeRootCmd(deps ...*Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vox-actor",
		Short:   "テキストをVOICEVOXエンジンで音声合成し読み上げるCLIツール",
		Version: version,
	}
	cmd.SilenceUsage = true
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}

	var d *Deps
	if len(deps) > 0 {
		d = deps[0]
	}

	var actDeps *ActDeps
	var watchDeps *WatchDeps
	var sayDeps *SayDeps
	var scriptAppendDeps *ScriptAppendDeps
	var scriptWriteDeps *ScriptWriteDeps
	var audioCheckDeps *AudioCheckDeps
	var viewerCheckDeps *ViewerCheckDeps
	var assetsDeps *AssetsDownloadDeps
	var speakersDeps *SpeakersDeps
	if d != nil {
		actDeps = d.Act
		watchDeps = d.Watch
		sayDeps = d.Say
		scriptAppendDeps = d.ScriptAppend
		scriptWriteDeps = d.ScriptWrite
		audioCheckDeps = d.AudioCheck
		viewerCheckDeps = d.ViewerCheck
		assetsDeps = d.Assets
		speakersDeps = d.Speakers
	}
	var viewerDeps *ViewerDeps
	if d != nil {
		viewerDeps = d.Viewer
	}
	cmd.AddCommand(makeActCmd(actDeps))
	cmd.AddCommand(makeWatchCmd(watchDeps))
	cmd.AddCommand(makeSayCmd(sayDeps))
	cmd.AddCommand(makeScriptCmd(scriptAppendDeps, scriptWriteDeps))
	cmd.AddCommand(makeConfigCmd())
	cmd.AddCommand(makeAudioCheckCmd(audioCheckDeps))
	cmd.AddCommand(makeViewerCheckCmd(viewerCheckDeps))
	cmd.AddCommand(makeViewerCmd(viewerDeps))
	cmd.AddCommand(makeAssetsCmd(assetsDeps))
	cmd.AddCommand(makeSpeakersCmd(speakersDeps))

	return cmd
}

// Execute はルートコマンドを実行する。
func Execute(deps *Deps) error {
	return makeRootCmd(deps).Execute()
}
