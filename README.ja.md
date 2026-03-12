# deck-pb

[k1LoW/deck](https://github.com/k1LoW/deck) で作成した Google Slides プレゼンテーションにプログレスバーを追加する CLI ツール  

## インストール

```bash
go install github.com/yashikota/deck-pb@latest
```

## 使い方

```bash
# プログレスバーを追加
deck-pb apply slides.md

# プログレスバーを削除
deck-pb delete slides.md
```

## 設定

`deck-pb.yml` にプログレスバーの見た目を設定します。

```yaml
progress:
  position: "bottom"   # "top" or "bottom" (default: "bottom")
  height: 10           # ピクセル (default: 10)
  color: "#4285F4"   # hex color (default: "#4285F4")
  startPage: 1         # 表示開始ページ (default: 1)
  endPage: 0           # 表示終了ページ, 0 = 最終ページ (default: 0)
```

別の設定ファイルを使う場合は `--config` フラグで指定できます。

```bash
deck-pb apply slides.md --config custom.yml
```

## 認証

deck と同じ認証情報 (`~/.local/share/deck/credentials.json`) を使用します。deck のセットアップが完了していればそのまま動作します。

環境変数による認証にも対応しています。

| 環境変数 | 説明 |
|---|---|
| `DECK_SERVICE_ACCOUNT_KEY` | サービスアカウントキーの JSON |
| `DECK_ENABLE_ADC` | Application Default Credentials を使用 |
| `DECK_ACCESS_TOKEN` | OAuth2 アクセストークン |

deck の profile 機能を使っている場合は `--profile` フラグで指定できます。

```bash
deck-pb apply slides.md --profile work
```
