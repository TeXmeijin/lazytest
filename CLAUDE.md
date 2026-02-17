# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

lazytest は PHPUnit テスト向けの TUI テストランナー。Go + Bubble Tea で構築され、TeamCity 形式の出力をリアルタイムにパースしてインタラクティブに表示する。

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

# 実行（カレントディレクトリの .lazytest.yml または phpunit.xml を自動検出）
./lazytest
```

## Architecture

### レイヤー構成

```
cmd/lazytest/main.go    エントリーポイント
internal/
  config/     設定読み込み（.lazytest.yml / phpunit.xml 自動検出）
  discovery/  テストファイルのスキャン（glob パターンマッチ）
  domain/     ドメイン型（TestStatus, TestCase, TestSuite, TestRun, TestFile）
  parser/     TeamCity 形式のストリーミングパーサー
  runner/     テスト実行（コマンドテンプレート展開 + ゴルーチン）
  ui/         Bubble Tea UI（SearchMode → RunningMode → ResultsMode）
```

### 処理フロー

1. `config.Load()` で設定読み込み（YAML優先、phpunit.xml フォールバック）
2. `discovery.ScanFiles()` でテストファイル一覧を取得
3. UI の SearchMode でファイル選択 → RunningMode で実行開始
4. `executor.Run()` がゴルーチンでコマンド実行し、stdout を `parser.ParseStream()` で TeamCity イベントに変換
5. イベントをチャネル経由で UI に送り、リアルタイム更新
6. ResultsMode で結果表示（左: スイート/テスト一覧、右: 詳細）

### コマンドテンプレート

設定の `command` フィールドで `{files}`（全ファイル）/ `{file}`（単一ファイル）がテストファイルパスに展開される。`path_strip_prefix` でパスのプレフィックスを除去。

### UI モード遷移

- **SearchMode**: テストファイル検索・選択 → Enter で RunningMode へ
- **RunningMode**: テスト実行中のリアルタイム進捗表示 → 完了で ResultsMode へ
- **ResultsMode**: 結果表示、`r` で再実行、`o` でエディタ起動、`f` で失敗フィルタ → Enter/Esc で SearchMode へ

### 主要な依存ライブラリ

- `charmbracelet/bubbletea`: TUI フレームワーク（Elm アーキテクチャ）
- `charmbracelet/lipgloss`: ターミナルスタイリング
- `charmbracelet/bubbles`: TUI コンポーネント（viewport 等）
- `gopkg.in/yaml.v3`: YAML パース
