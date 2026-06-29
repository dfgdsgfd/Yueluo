function checkFrontendPublishUploadStaticContract(checks) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
    frontendTypesPath: path.relative(repoRoot, frontendTypesPath),
    frontendPublishWorkbenchPath: path.relative(repoRoot, frontendPublishWorkbenchPath),
    backendConfigPath: path.relative(repoRoot, backendConfigPath),
    backendUploadHandlerPath: path.relative(repoRoot, backendUploadHandlerPath),
    backendFileHandlerPath: path.relative(repoRoot, backendFileHandlerPath),
    backendFileSigningHandlerPath: path.relative(repoRoot, backendFileSigningHandlerPath),
    backendFileHandlerTestPath: path.relative(repoRoot, backendFileHandlerTestPath),
    backendEnvExamplePath: path.relative(repoRoot, backendEnvExamplePath),
  };

  const frontendApi = fileText(frontendApiPath);
  const frontendTypes = fileText(frontendTypesPath);
  const publishWorkbench = fileText(frontendPublishWorkbenchPath);
  const backendConfig = fileText(backendConfigPath);
  const backendUploadHandler = fileText(backendUploadHandlerPath);
  const backendFileHandler = fileText(backendFileHandlerPath);
  const backendFileSigningHandler = fileText(backendFileSigningHandlerPath);
  const backendFileHandlerTest = fileText(backendFileHandlerTestPath);
  const backendEnvExample = fileText(backendEnvExamplePath);

  try {
    const matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
    const routes = Array.isArray(matrix.routes) ? matrix.routes.filter(isRecord) : [];
    const expectedRoutes = [
      { method: "POST", path: "/api/upload/single", auth: "user" },
      { method: "POST", path: "/api/upload/video", auth: "user" },
      { method: "POST", path: "/api/upload/attachment", auth: "user" },
      { method: "POST", path: "/api/posts", auth: "user" },
    ];
    const routeResults = [];

    for (const expected of expectedRoutes) {
      const matches = routes.filter(
        (route) => route.method === expected.method && route.path === expected.path,
      );
      const actual = matches[0] ?? null;
      const result = {
        method: expected.method,
        path: expected.path,
        expectedAuth: expected.auth,
        actualAuth: actual?.auth ?? null,
        actualStatus: actual?.status ?? null,
        ok:
          matches.length === 1 &&
          actual?.auth === expected.auth &&
          actual?.status === "native-gin",
      };
      routeResults.push(result);
      if (!result.ok) {
        problems.push(`${expected.method} ${expected.path} route auth/status is not aligned`);
      }
    }

    details.publishUploadRoutes = routeResults;
  } catch (error) {
    problems.push(`backend route matrix cannot be read: ${error.message}`);
  }

  const requiredPatterns = [
    {
      fileText: frontendApi,
      pattern: "export async function uploadImage",
      problem: "frontend uploadImage helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiUpload<UploadAsset>("/api/upload/single", formData, options)',
      problem: "frontend image upload does not call /api/upload/single",
    },
    {
      fileText: frontendApi,
      pattern: 'uploadFileInChunks(file, "/api/upload/chunk/merge/image", options)',
      problem: "frontend image upload does not use chunk merge image for large files",
    },
    {
      fileText: frontendApi,
      pattern: "export async function uploadVideo",
      problem: "frontend uploadVideo helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'formData.set("thumbnail", thumbnail)',
      problem: "frontend video upload does not include optional thumbnail form field",
    },
    {
      fileText: frontendApi,
      pattern: 'apiUpload<UploadAsset>("/api/upload/video", formData, options)',
      problem: "frontend video upload does not call /api/upload/video",
    },
    {
      fileText: frontendApi,
      pattern: 'uploadFileInChunks(file, "/api/upload/chunk/merge"',
      problem: "frontend video upload does not use chunk merge for large files",
    },
    {
      fileText: frontendApi,
      pattern: '"/api/upload/chunk/config"',
      problem: "frontend chunk upload config endpoint is missing",
    },
    {
      fileText: frontendApi,
      pattern: '"/api/upload/chunk/verify"',
      problem: "frontend chunk upload verify endpoint is missing",
    },
    {
      fileText: frontendApi,
      pattern: '"/api/upload/chunk"',
      problem: "frontend chunk upload endpoint is missing",
    },
    {
      fileText: frontendApi,
      pattern: "export async function uploadAttachment",
      problem: "frontend uploadAttachment helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiUpload<UploadAsset>("/api/upload/attachment", formData, options)',
      problem: "frontend attachment upload does not call /api/upload/attachment",
    },
    {
      fileText: frontendApi,
      pattern: "export function createPost(input: PublishPostInput)",
      problem: "frontend createPost helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<{ id: number }>("/api/posts", input)',
      problem: "frontend createPost does not call /api/posts",
    },
    {
      fileText: publishWorkbench,
      pattern: 'type PublishMode = "video" | "image" | "article" | "podcast"',
      problem: "publish workbench does not expose the expected publish modes",
    },
    {
      fileText: publishWorkbench,
      pattern: 'podcast: "audio/*"',
      problem: "publish workbench podcast mode does not restrict file picker to audio/*",
    },
    {
      fileText: publishWorkbench,
      pattern: "podcast: 1024 * 1024 * 1024",
      problem: "publish workbench podcast size limit is missing",
    },
    {
      fileText: publishWorkbench,
      pattern: "await createPost(payload)",
      problem: "publish workbench does not submit via createPost",
    },
    {
      fileText: frontendTypes,
      pattern: "signedUrl?: string",
      problem: "frontend UploadAsset type does not expose signedUrl",
    },
    {
      fileText: frontendTypes,
      pattern: "coverSignedUrl?: string | null",
      problem: "frontend UploadAsset type does not expose coverSignedUrl",
    },
    {
      fileText: frontendApi,
      pattern: "coverSignedUrl: cover.signedUrl ?? cover.url",
      problem: "frontend video upload does not preserve signed cover URL",
    },
    {
      fileText: publishWorkbench,
      pattern: "return asset.signedUrl || asset.url",
      problem: "publish workbench previews do not prefer signed upload URLs",
    },
    {
      fileText: publishWorkbench,
      pattern: "return asset.coverSignedUrl || asset.coverUrl || null",
      problem: "publish workbench video previews do not prefer signed cover URLs",
    },
    {
      fileText: publishWorkbench,
      pattern: "uploadImage(file, { onProgress })",
      problem: "publish workbench image mode does not reuse uploadImage with progress",
    },
    {
      fileText: publishWorkbench,
      pattern: "uploadVideo(file, thumbnail, { onProgress })",
      problem: "publish workbench video mode does not reuse uploadVideo with progress",
    },
    {
      fileText: publishWorkbench,
      pattern: "uploadAttachment(file, { onProgress })",
      problem: "publish workbench podcast mode does not reuse uploadAttachment with progress",
    },
    {
      fileText: publishWorkbench,
      pattern: "input.attachment = {",
      problem: "publish workbench podcast mode does not map uploaded audio to attachment",
    },
    {
      fileText: publishWorkbench,
      pattern: 'filename: audio.originalname ?? "audio"',
      problem: "publish workbench podcast attachment does not preserve filename",
    },
    {
      fileText: publishWorkbench,
      pattern: "filesize: audio.size ?? 0",
      problem: "publish workbench podcast attachment does not preserve filesize",
    },
    {
      fileText: publishWorkbench,
      pattern: 'type: mode === "video" ? 2 : 1',
      problem: "publish workbench post type mapping for video/non-video is missing",
    },
    {
      fileText: backendUploadHandler,
      pattern: "func (h NativeHandlers) UploadAttachment",
      problem: "backend UploadAttachment handler is missing",
    },
    {
      fileText: backendUploadHandler,
      pattern: 'allowedAttachmentType(file.Header.Get("Content-Type"))',
      problem: "backend attachment upload does not validate attachment MIME type",
    },
    {
      fileText: backendUploadHandler,
      pattern: 'h.Config.Upload.Attachment.LocalUploadDir, "attachments", file.Filename',
      problem: "backend attachment upload does not store files in the attachments namespace",
    },
    {
      fileText: backendUploadHandler,
      pattern: 'strings.HasPrefix(value, "audio/")',
      problem: "backend attachment MIME allow-list does not allow audio/*",
    },
    {
      fileText: backendUploadHandler,
      pattern: '"application/pdf"',
      problem: "backend attachment MIME allow-list does not preserve document uploads",
    },
    {
      fileText: backendUploadHandler,
      pattern: '"signedUrl":',
      problem: "backend upload responses do not include signedUrl",
    },
    {
      fileText: backendUploadHandler,
      pattern: '"coverSignedUrl":',
      problem: "backend video upload responses do not include coverSignedUrl",
    },
    {
      fileText: backendUploadHandler,
      pattern: 'return "/api/file/" + urlType + "/" + unique',
      problem: "backend local uploads do not store canonical /api/file paths",
    },
    {
      fileText: backendUploadHandler,
      pattern: 'return outPath, "/api/file/" + urlType + "/" + unique',
      problem: "backend chunk merges do not store canonical /api/file paths",
    },
    {
      fileText: backendConfig,
      pattern: "FileSigning UploadFileSigningConfig",
      problem: "backend config does not expose upload file signing settings",
    },
    {
      fileText: backendConfig,
      pattern: 'getEnv("FILE_SIGNING_SECRET", getEnv("JWT_SECRET"',
      problem: "backend file signing secret does not default from JWT_SECRET",
    },
    {
      fileText: backendConfig,
      pattern: 'intEnv("FILE_SIGNING_TTL_SECONDS", 15*60)',
      problem: "backend file signing TTL env default is missing",
    },
    {
      fileText: backendEnvExample,
      pattern: "FILE_SIGNING_SECRET=",
      problem: "backend .env.example does not document FILE_SIGNING_SECRET",
    },
    {
      fileText: backendEnvExample,
      pattern: "FILE_SIGNING_TTL_SECONDS=900",
      problem: "backend .env.example does not document FILE_SIGNING_TTL_SECONDS",
    },
    {
      fileText: backendFileHandler,
      pattern: "h.verifyFileURLSignature(canonicalPath, c.Query(fileSignatureExpiryParam), c.Query(fileSignatureParam), time.Now())",
      problem: "backend /api/file access does not accept short signed URLs",
    },
    {
      fileText: backendFileSigningHandler,
      pattern: 'fileSignatureExpiryParam = "pvimg_exp"',
      problem: "backend file signing does not use pvimg_exp expiry parameter",
    },
    {
      fileText: backendFileSigningHandler,
      pattern: 'fileSignatureParam       = "sign"',
      problem: "backend file signing does not use sign parameter",
    },
    {
      fileText: backendFileSigningHandler,
      pattern: "hmac.New(sha256.New",
      problem: "backend file signing does not use HMAC-SHA256",
    },
    {
      fileText: backendFileSigningHandler,
      pattern: "base64.RawURLEncoding.EncodeToString",
      problem: "backend file signing does not emit base64url signatures",
    },
    {
      fileText: backendFileSigningHandler,
      pattern: "func (h NativeHandlers) normalizeFileURLForStorage",
      problem: "backend signed file URL storage normalization is missing",
    },
    {
      fileText: backendFileSigningHandler,
      pattern: "func (h NativeHandlers) verifyFileURLSignature",
      problem: "backend signed file URL verification helper is missing",
    },
    {
      fileText: backendFileHandlerTest,
      pattern: "TestAllowedAttachmentTypeAcceptsAudio",
      problem: "backend tests do not cover audio attachment MIME allow-list",
    },
    {
      fileText: backendFileHandlerTest,
      pattern: 'allowedAttachmentType("video/mp4")',
      problem: "backend tests do not assert video/mp4 is rejected as an attachment",
    },
    {
      fileText: backendFileHandlerTest,
      pattern: "TestFileAccessAcceptsShortSignatureWhenGuestRestricted",
      problem: "backend tests do not cover short signed /api/file access",
    },
    {
      fileText: backendFileHandlerTest,
      pattern: "TestFileAccessRejectsMissingExpiredAndTamperedSignatureWhenGuestRestricted",
      problem: "backend tests do not cover rejected missing/expired/tampered file signatures",
    },
    {
      fileText: backendFileHandlerTest,
      pattern: "TestNormalizeFileURLForStorageStripsLocalSignature",
      problem: "backend tests do not cover signed local file URL normalization",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  addCheck(
    checks,
    "frontend-publish-upload-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend publish/upload contract is aligned with backend upload routes"
      : "frontend publish/upload contract is not aligned with backend upload routes",
    {
      ...details,
      problems,
    },
  );
}

