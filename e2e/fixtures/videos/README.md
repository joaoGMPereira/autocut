# E2E Video Fixtures

This directory holds YouTube videos used by the E2E suite. The videos are **not**
committed — download them locally with:

```sh
pnpm --filter @autocut/e2e fixtures
# or, from the AutoCut root:
sh e2e/fixtures/download-fixtures.sh
```

The script downloads three videos at 720p mp4:

| Video ID       | Source URL                                        |
|----------------|---------------------------------------------------|
| bfJy1-IRa_k    | https://www.youtube.com/watch?v=bfJy1-IRa_k       |
| HsNMliaypC0    | https://www.youtube.com/watch?v=HsNMliaypC0       |
| 7hyc3z2WSkQ    | https://www.youtube.com/watch?v=7hyc3z2WSkQ       |

Files are named `{video_id}.mp4`. They're gitignored.
