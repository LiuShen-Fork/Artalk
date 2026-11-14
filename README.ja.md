<p align="center">
<img src="https://user-images.githubusercontent.com/22412567/171680920-6e74b77c-c565-487b-bff1-4f94976ecbe7.png" alt="Artalk" width="100%">
</p>

# Artalk Fork

[![npm version](https://img.shields.io/npm/v/artalk.svg?style=flat-square)](https://www.npmjs.com/package/artalk)
[![npm downloads](https://img.shields.io/npm/dt/artalk.svg?style=flat-square)](https://www.npmjs.com/package/artalk)
[![Docker Pulls](https://img.shields.io/docker/pulls/artalk/artalk-go?style=flat-square)](https://hub.docker.com/r/artalk/artalk-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/artalkjs/artalk/v2.svg)](https://pkg.go.dev/github.com/artalkjs/artalk/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/ArtalkJS/Artalk?style=flat-square)](https://goreportcard.com/report/github.com/ArtalkJS/Artalk)
[![CircleCI](https://img.shields.io/circleci/build/gh/ArtalkJS/Artalk?style=flat-square)](https://circleci.com/gh/ArtalkJS/Artalk/tree/master)
[![Codecov](https://img.shields.io/codecov/c/gh/ArtalkJS/Artalk?style=flat-square)](https://codecov.io/gh/ArtalkJS/Artalk)
[![npm bundle size](https://img.shields.io/bundlephobia/minzip/artalk?style=flat-square)](https://bundlephobia.com/package/artalk)

[ホームページ](https://artalk.js.org) • [ドキュメント](https://artalk.js.org/en/guide/deploy.html) • [最新リリース](https://github.com/ArtalkJS/Artalk/releases) • [変更履歴](https://github.com/ArtalkJS/Artalk/blob/master/CHANGELOG.md) • [English](./README.md) • [简体中文](./README.zh.md)

Artalk は直感的かつ高機能なコメントシステムで、あらゆるブログ、ウェブサイト、ウェブアプリケーションにすぐに導入できます。

- 🍃 クライアントは約 40KB、純粋な Vanilla JS で作られており、フレームワークに依存しません
- 🍱 サーバーは Golang を採用し、効率的かつ軽量なクロスプラットフォーム性能を提供します
- 🐳 Docker によるワンクリックデプロイで、手軽かつ高速に導入できます
- 🌈 オープンソースソフトウェアで、プライバシーを最優先にセルフホストできます

このリポジトリは [ArtalkJS/Artalk](https://github.com/ArtalkJS/Artalk) をベースにした個人メンテナンスの Fork です。上流のクライアント API と基本的なコメント機能との互換性を維持し、バックエンド、管理コンソール、セルフホスト運用向けの機能を中心に変更しています。

## Fork と上流リポジトリ

- 上流リポジトリ: [ArtalkJS/Artalk](https://github.com/ArtalkJS/Artalk)
- 現在のリポジトリ: [LiuShen-Fork/Artalk](https://github.com/LiuShen-Fork/Artalk)
- 上流ドキュメント: [artalk.js.org](https://artalk.js.org)
- `master` は `upstream/master` をベースとしており、Fork 固有のコミットはこのリポジトリに保持して手動で同期しています。
- 独自の npm クライアントパッケージは公開していません。フロントエンドクライアントは上流版をベースとし、サーバーと管理コンソールをこのリポジトリで保守しています。

## 上流との差分

### AI コメントモデレーション

既存のモデレーション処理に、任意で利用できる非同期 AI モデレーションを追加しています。OpenAI Responses、OpenAI Chat Completions、DeepSeek JSON Output に対応し、JSON Schema 出力、編集可能なプロンプト、思考モード、API リクエスト失敗時の処理を設定できます。

### AI コメントアシスタント

設定した名前でメンションされたコメントに返信する、任意利用の AI アシスタントを追加しています。ページ内容、最近のコメント、現在のコメントを文脈として利用し、ネストされたコメントでの重複返信を避けます。OpenAI Responses、OpenAI Chat Completions、DeepSeek JSON Output、Claude Messages に対応しています。`reply_to_pending` で、モデレーション待ちのコメントから返信を生成するかどうかを制御できます。プロンプトは管理コンソールの複数行入力欄から編集できます。

### その他の Fork 固有の変更

- モデレーションと AI アシスタントのログ、レビュー操作、ダッシュボード、中文をデフォルトとした管理コンソール。
- 汎用 OAuth 2.0 ログインと OIDC SSO トークン交換。
- Upgit を残したまま、Lsky Pro への直接画像アップロードに対応。
- IPv6 用 ip2region データベースを個別に設定可能。
- Fork 向けの CI/CD と Docker 公開ワークフロー。リリースは手動実行です。

Markdown、ネストされたコメント、マルチサイト、通知、CAPTCHA、キーワードモデレーション、各種データベース、Docker、ローカライズ、ダークモード、遅延読み込み、ライトボックス、IP 地域表示など、上流の基本機能は引き続き利用できます。

## 上流との同期

~~~bash
git remote add upstream https://github.com/ArtalkJS/Artalk.git
git fetch upstream
git rebase upstream/master
~~~

rebase 後は、モデレーション処理、設定メタデータ、管理コンソール、リリースワークフローの競合を確認し、バックエンドのテストと管理コンソールのビルドを実行してください。

## 機能

<!-- prettier-ignore-start -->

<!-- features -->
* [サイドバー](https://artalk.js.org/guide/frontend/sidebar.html): 素早い管理と直感的な閲覧
* [ソーシャルログイン](https://artalk.js.org/guide/frontend/auth.html): ソーシャルアカウントによる高速ログイン
* [メール通知](https://artalk.js.org/guide/backend/email.html): 多様な送信方法とメールテンプレート
* [多様なプッシュ通知](https://artalk.js.org/guide/backend/admin_notify.html): 複数のプッシュ方法と通知テンプレート
* [サイト内通知](https://artalk.js.org/guide/frontend/sidebar.html): 未読マークとメンション一覧
* [キャプチャ](https://artalk.js.org/guide/backend/captcha.html): 多様な認証タイプと頻度制限
* [コメントモデレーション](https://artalk.js.org/guide/backend/moderator.html): コンテンツ検出とスパム遮断
* [画像アップロード](https://artalk.js.org/guide/backend/img-upload.html): カスタムアップロード、画像ホスティング対応
* [Markdown](https://artalk.js.org/guide/intro.html): Markdown 構文に対応
* [絵文字パック](https://artalk.js.org/guide/frontend/emoticons.html): OwO 互換、簡単に統合
* [マルチサイト](https://artalk.js.org/guide/backend/multi-site.html): サイトの分離と一元管理
* [管理者](https://artalk.js.org/guide/backend/multi-site.html): パスワード認証とバッジ識別
* [ページ管理](https://artalk.js.org/guide/frontend/sidebar.html): 素早い閲覧とワンクリックでのタイトル遷移
* [ページビュー統計](https://artalk.js.org/guide/frontend/pv.html): ページビューを簡単に追跡
* [階層構造](https://artalk.js.org/guide/frontend/config.html#nestmax): ネストされたページネーション一覧と無限スクロール
* [コメント投票](https://artalk.js.org/guide/frontend/config.html#vote): コメントの賛成・反対投票
* [コメント並び替え](https://artalk.js.org/guide/frontend/config.html#listsort): 多様な並び替えオプションを自由に選択
* [コメント検索](https://artalk.js.org/guide/frontend/sidebar.html): コメント内容を素早く検索
* [コメントのピン留め](https://artalk.js.org/guide/frontend/sidebar.html): 重要なメッセージをピン留め
* [作成者のみ表示](https://artalk.js.org/guide/frontend/config.html): 作成者のコメントのみを表示
* [コメントジャンプ](https://artalk.js.org/guide/intro.html): 引用されたコメントへ素早く移動
* [自動保存](https://artalk.js.org/guide/frontend/config.html): 入力内容の消失を防止
* [IP 地域表示](https://artalk.js.org/guide/frontend/ip-region.html): ユーザーの IP 地域を表示
* [データ移行](https://artalk.js.org/guide/transfer.html): 自由な移行と素早いバックアップ
* [画像ライトボックス](https://artalk.js.org/guide/frontend/lightbox.html): 画像ライトボックスを簡単に統合
* [画像の遅延読み込み](https://artalk.js.org/guide/frontend/img-lazy-load.html): 画像を遅延読み込みし、体験を最適化
* [LaTeX](https://artalk.js.org/guide/frontend/latex.html): LaTeX 数式の解析を統合
* [ナイトモード](https://artalk.js.org/guide/frontend/config.html#darkmode): ナイトモードへの切り替え
* [拡張プラグイン](https://artalk.js.org/develop/plugin.html): さらなる可能性を創出
* [多言語対応](https://artalk.js.org/guide/frontend/i18n.html): 複数の言語を切り替え
* [コマンドライン](https://artalk.js.org/guide/backend/config.html): コマンドラインによる操作管理
* [API ドキュメント](https://artalk.js.org/http-api.html): OpenAPI 形式のドキュメントを提供
* [プログラムのアップグレード](https://artalk.js.org/guide/backend/update.html): バージョンチェックとワンクリックアップグレード
<!-- /features -->

<!-- prettier-ignore-end -->

## インストール

Docker でワンステップで Artalk サーバーをデプロイ:

```bash
docker run -d \
    --name artalk \
    -p 8080:23366 \
    -v $(pwd)/data:/data \
    -e "TZ=America/New_York" \
    -e "ATK_LOCALE=en" \
    -e "ATK_SITE_DEFAULT=Artalk Blog" \
    -e "ATK_SITE_URL=https://example.com" \
    artalk/artalk-go
```

Artalk クライアントをウェブページに組み込む:

<!-- prettier-ignore-start -->

```ts
Artalk.init({
  el:      '#Comments',
  site:    'Artalk Blog',
  server:  'https://artalk.example.com',
  pageKey: '/2018/10/02/hello-world.html'
})
```

<!-- prettier-ignore-end -->

バイナリファイル、go install、Linux ディストリビューション向けのパッケージマネージャーなど、さまざまなインストール方法を提供しています。

[**詳しく見る →**](https://artalk.js.org/en/guide/deploy.html)

## 開発者の方へ

プルリクエストを歓迎します！

コードベースの扱い方、ローカル開発環境のセットアップ、変更の貢献については [開発](https://artalk.js.org/en/develop/) と [コントリビューション](./CONTRIBUTING.md) をご覧ください。

## コントリビューター

皆さんの貢献はオープンソースコミュニティを豊かにし、学び、ひらめき、イノベーションを育みます。私たちは皆さんの参加を心から大切にしています。私たちのコミュニティに欠かせない存在でいてくれて、ありがとうございます！🥰

[![](https://contrib.rocks/image?repo=ArtalkJS/Artalk)](https://github.com/ArtalkJS/Artalk/graphs/contributors)

## サポーター

[![Stargazers repo roster for @ArtalkJS/Artalk](https://reporoster.com/stars/ArtalkJS/Artalk)](https://github.com/ArtalkJS/Artalk/stargazers)

## Repobeats アナリティクス

![Alt](https://repobeats.axiom.co/api/embed/a9fc9191ac561bc5a8ee2cddc81e635ecaebafb6.svg 'Repobeats analytics image')

## Stargazer の推移

<a href="https://trendshift.io/repositories/6290" target="_blank"><img src="https://trendshift.io/api/badge/repositories/6290" alt="ArtalkJS%2FArtalk | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[![Stargazers over time](https://starchart.cc/ArtalkJS/Artalk.svg)](https://starchart.cc/ArtalkJS/Artalk)

## ライセンス

[MIT](./LICENSE)

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk?ref=badge_shield)

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FArtalkJS%2FArtalk?ref=badge_large)
