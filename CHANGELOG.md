# Changelog

## Unreleased

### Breaking Changes

#### 画像配置先ディレクトリを `.vox-actor/characters/` から `.vox-actor/assets/` へ変更

キャラクター画像の配置先ディレクトリが変更されました。

**変更前:**
```
.vox-actor/
└── characters/
    ├── zundamon_closed.png
    └── zundamon_open.png
```

**変更後:**
```
.vox-actor/
└── assets/
    ├── zundamon_closed.png
    └── zundamon_open.png
```

**マイグレーション手順:**

既存の `.vox-actor/characters/` ディレクトリを `.vox-actor/assets/` へ移動してください。

```bash
mv .vox-actor/characters .vox-actor/assets
```

また、画像配信エンドポイントも変更されています:

- 変更前: `/assets/images/characters/<filename>`
- 変更後: `/assets/images/<filename>`
