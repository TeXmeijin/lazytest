# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

lazytest はマルチフレームワーク対応の TUI テストランナー。Go + Bubble Tea で構築され、TeamCity 形式の出力をリアルタイムにパースしてインタラクティブに表示する。PHPUnit、Vitest 等を単一 TUI から横断実行できる。

## Build & Test Commands

```bash
# ビルド
go build -o lazytest ./cmd/lazytest

# テスト全実行
go test ./...

# 特定パッケージのテスト
go test ./internal/config
go test ./internal/parser
go test ./internal/discovery
go test ./internal/runner

# 特定テスト関数の実行
go test -run TestFuncName ./internal/parser

# カバレッジ
go test -cover ./...

# 実行（カレントディレクトリの .lazytest.yml またはフレームワーク設定ファイルを自動検出）
./lazytest
```

## Architecture

### レイヤー構成

```
cmd/lazytest/main.go    エントリーポイント
internal/
  config/     設定読み込み（.lazytest.yml / フレームワーク自動検出）
    config.go   Target + Config 構造体、Load()、FindProjectRoot()
    detect.go   DetectFrameworks()（PHPUnit / Vitest 自動検出、最大3階層走査）
    phpunit.go  phpunit.xml パーサー
  discovery/  テストファイルのスキャン
    scanner.go  ScanFiles()（カンマ区切りパターン対応）、ScanAllTargets()
  domain/     ドメイン型（TestFile, TestCase, TestSuite, TestRun, AggregatedRun）
  parser/     TeamCity 形式のストリーミングパーサー（PHPUnit/Vitest 両対応）
  runner/     マルチターゲット並列実行（TargetEvent, ゴルーチン + fan-in）
  ui/         Bubble Tea UI（SearchMode → RunningMode → ResultsMode）
```

### 処理フロー

1. `config.Load()` で設定読み込み（YAML優先、`DetectFrameworks()` フォールバック）
2. `discovery.ScanAllTargets()` で全ターゲットのテストファイル一覧を取得
3. UI の SearchMode でファイル選択（ターゲットバッジ付き）→ RunningMode で実行開始
4. `executor.Run()` がファイルを TargetName でグルーピングし、ターゲットごとにゴルーチンで並列実行
5. 各ゴルーチンが stdout を `parser.ParseStream()` で TeamCity イベントに変換、`TargetEvent` でラップして共有チャネルへ fan-in
6. UI が `TargetEvent` を受けてターゲット別にリアルタイム更新
7. 全ターゲット完了後、`AggregatedRun` を構築して ResultsMode で統合結果表示

### 設定構造

```yaml
editor: zed
targets:
  - name: phpunit          # ターゲット名（デフォルト値の決定に使用）
    command: "..."         # {files} / {file} がテストパスに展開される
    test_dirs: [...]       # スキャン対象ディレクトリ
    file_pattern: "..."    # カンマ区切りで OR マッチ可能
    path_strip_prefix: ""  # コマンドに渡す前にパスから除去
    working_dir: ""        # コマンド実行ディレクトリ（パスも自動調整）
```

### UI モード遷移

- **SearchMode**: テストファイル検索・選択（ターゲットバッジ付き）→ Enter で RunningMode へ
- **RunningMode**: ターゲット別リアルタイム進捗表示 → 全完了で ResultsMode へ
- **ResultsMode**: 統合結果表示（ターゲットバッジ付き）、`r` で再実行、`o` でエディタ起動、`f` で失敗フィルタ → Enter/Esc で SearchMode へ

### 主要な依存ライブラリ

- `charmbracelet/bubbletea`: TUI フレームワーク（Elm アーキテクチャ）
- `charmbracelet/lipgloss`: ターミナルスタイリング
- `charmbracelet/bubbles`: TUI コンポーネント（viewport 等）
- `gopkg.in/yaml.v3`: YAML パース
