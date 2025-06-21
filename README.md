# txoutproof-graph

`txoutproof-graph` は、Goで書かれたコマンドラインツールです。BitcoinのPartial Merkle Tree（部分マークルツリー、`merkleblock` データとしても知られています）を解析し、Graphvizを使ってツリー構造を可視化します。

このツールは、16進数文字列を引数として受け取り、それをブロックヘッダーとマークルプルーフにデコードします。その後、部分マークルツリーの画像を描画するために使用できるDOT言語の文字列を出力します。

## 機能

*   80バイトのBitcoinブロックヘッダーを解析
*   BIP 37で定義された部分マークルツリーのデータを解析
*   可変長整数フォーマット（CompactSize）のデコード
*   提供されたハッシュとフラグを使用して部分マークルツリーを再構築
*   ツリーのGraphviz DOT言語表現を生成
*   可視化において、リーフノード（葉）と内部ブランチノード（枝）を区別して表示

## 必要なもの

このツールを使用する前に、以下のソフトウェアがインストールされている必要があります。

1.  **Go**: バージョン 1.22 以降
2.  **Graphviz**: DOT言語の出力を画像に変換するために必要です。
    *   macOSの場合: `brew install graphviz`
    *   Debian/Ubuntuの場合: `sudo apt-get install graphviz`

## インストール

1.  リポジトリをクローンします。
    ```sh
    git clone <your-repo-url>
    cd txoutproof-graph
    ```

2.  Goの依存関係をインストールします。
    ```sh
    go mod tidy
    ```

## 使い方

1.  プログラムをビルドします。
    ```sh
    go build
    ```

2.  `merkleblock` メッセージに対応する16進数文字列を引数としてプログラムを実行します。出力は `tree.dot` ファイルにリダイレクトされます。
    ```sh
    ./txoutproof-graph <merkleblock_hex_string> > tree.dot
    ```

3.  `dot` コマンドを使用して、`.dot` ファイルを画像（例: PNG）に変換します。
    ```sh
    dot -Tpng tree.dot -o tree.png
    ```

    `tree.png` を開くと、部分マークルツリーの可視化された画像が表示されます。

### 実行例

以下は、特定のトランザクションを含むブロックの `merkleblock` データの例です。

```sh
# 実際のマークルブロックの16進数文字列に置き換えてください
HEX_STRING="0100000043497fd7f826957108f4a30fd9cec3aeba79972084e90ead01ea330900000000bac8b0fa92da48d597504b7ea5554272f9d13a7f118760c4e01dc43f5a0f00002E43104D4653C3F321685202000000000101000000010000000000000000000000000000000000000000000000000000000000000000ffffffff08044d10432e0102ffffffff0100f2052a0100000043410479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8ac00000000"

./txoutproof-graph $HEX_STRING > tree.dot

dot -Tpng tree.dot -o example_tree.png
```

!Example Merkle Tree

*(注: 上記の画像を表示するには、実際にコマンドを実行して `example_tree.png` を生成する必要があります。)*

## コードの構造

*   `main.go`: メインのアプリケーションロジック。コマンドライン引数の解析、データのデコード、ツリー構築の開始を行います。
*   **BlockHeader**: Bitcoinのブロックヘッダー構造を定義します。
*   **MerkleProofData**: ブロックヘッダーに続く部分マークルツリーのデータを定義します。
*   **buildAndDrawPartialTree**: `vbits` と `hashes` を元に再帰的に部分マークルツリーを構築し、Graphvizのノードとエッジを生成する中心的な関数です。

## ライセンス

このプロジェクトは Apache License 2.0 の下で公開されています。