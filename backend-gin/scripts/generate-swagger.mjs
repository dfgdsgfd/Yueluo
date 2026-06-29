import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ginRoot = path.resolve(__dirname, '..');
const matrixPath = path.join(ginRoot, 'internal', 'http', 'routes', 'route-matrix.json');
const outputPath = path.join(ginRoot, 'internal', 'http', 'swagger', 'swagger.json');

const swaggerDocsPath = process.env.SWAGGER_DOCS_PATH || 'swagger-MYQD6LuH0heYgcK5DT10Al00dj6OW8Wc';
const jwtTestTokenPath = process.env.JWT_TEST_TOKEN_PATH || 'jwt-MYQD6LuH0heYgcK5DT10Al00dj6OW8Wc';

const matrix = JSON.parse(fs.readFileSync(matrixPath, 'utf8'));

function materializePath(routePath) {
  return routePath
    .replaceAll('${SWAGGER_DOCS_PATH}', swaggerDocsPath)
    .replaceAll('${JWT_TEST_TOKEN_PATH}', jwtTestTokenPath);
}

function openAPIPath(routePath) {
  let wildcardIndex = 0;
  return materializePath(routePath)
    .split('/')
    .map((part) => {
      if (part.startsWith(':')) return `{${part.slice(1)}}`;
      if (part === '*') return `{wildcard${++wildcardIndex}}`;
      if (part.startsWith('*')) return `{${part.slice(1) || `wildcard${++wildcardIndex}`}}`;
      return part;
    })
    .join('/');
}

function pathParameters(pathTemplate) {
  const names = [...pathTemplate.matchAll(/\{([^}]+)\}/g)].map((match) => match[1]);
  return [...new Set(names)].map((name) => ({
    name,
    in: 'path',
    required: true,
    schema: { type: 'string' },
  }));
}

function tagFor(route) {
  const file = route.sourceFile || 'app';
  return file.replace(/^backend\/routes\//, '').replace(/^backend\//, '').replace(/\.js$/, '');
}

function methodList(method) {
  const normalized = method.toLowerCase();
  if (normalized === 'all') return ['get', 'post', 'put', 'delete', 'patch'];
  return [normalized];
}

function securityFor(route) {
  switch (route.auth) {
    case 'admin':
    case 'user':
      return [{ bearerAuth: [] }];
    case 'openapi':
      return [{ apiKeyAuth: [] }];
    default:
      return [];
  }
}

function operationId(method, pathTemplate) {
  return `${method}_${pathTemplate}`
    .replace(/[^A-Za-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .toLowerCase();
}

function genericOperation(route, method, pathTemplate) {
  const operation = {
    operationId: operationId(method, pathTemplate),
    summary: `${method.toUpperCase()} ${pathTemplate}`,
    description: [
      `Source: ${route.sourceFile}:${route.line}`,
      route.notes ? `Notes: ${route.notes}` : '',
    ].filter(Boolean).join('\n'),
    tags: [tagFor(route)],
    parameters: pathParameters(pathTemplate),
    responses: {
      200: {
        description: 'Successful response',
        content: {
          'application/json': {
            schema: { $ref: '#/components/schemas/SuccessResponse' },
          },
        },
      },
      400: {
        description: 'Bad request',
        content: {
          'application/json': {
            schema: { $ref: '#/components/schemas/ErrorResponse' },
          },
        },
      },
      401: {
        description: 'Unauthorized',
        content: {
          'application/json': {
            schema: { $ref: '#/components/schemas/ErrorResponse' },
          },
        },
      },
      500: {
        description: 'Server error',
        content: {
          'application/json': {
            schema: { $ref: '#/components/schemas/ErrorResponse' },
          },
        },
      },
    },
    'x-auth': route.auth,
    'x-source-file': route.sourceFile,
    'x-source-line': route.line,
  };
  const security = securityFor(route);
  if (security.length) operation.security = security;
  if (['post', 'put', 'patch', 'delete'].includes(method)) {
    operation.requestBody = {
      required: false,
      content: {
        'application/json': {
          schema: { type: 'object', additionalProperties: true },
        },
      },
    };
  }
  return operation;
}

const paths = {};
for (const route of matrix.routes) {
  const pathTemplate = openAPIPath(route.path);
  paths[pathTemplate] ||= {};
  for (const method of methodList(route.method)) {
    paths[pathTemplate][method] = genericOperation(route, method, pathTemplate);
  }
}

const spec = {
  openapi: '3.0.0',
  info: {
    title: 'Yuem Go-Gin API',
    version: 'gin-migration',
    description: 'Generated from the Go-Gin route matrix during build.',
  },
  servers: [{ url: '/', description: 'Current server' }],
  tags: [...new Set(matrix.routes.map(tagFor))].sort().map((name) => ({ name })),
  paths,
  components: {
    securitySchemes: {
      bearerAuth: {
        type: 'http',
        scheme: 'bearer',
        bearerFormat: 'JWT',
      },
      apiKeyAuth: {
        type: 'apiKey',
        in: 'header',
        name: 'X-API-Key',
      },
    },
    schemas: {
      SuccessResponse: {
        type: 'object',
        properties: {
          code: { type: 'integer', example: 200 },
          message: { type: 'string', example: 'success' },
          data: { type: 'object', nullable: true, additionalProperties: true },
        },
        additionalProperties: true,
      },
      ErrorResponse: {
        type: 'object',
        properties: {
          code: { type: 'integer', example: 500 },
          message: { type: 'string' },
          requestId: { type: 'string' },
          details: { nullable: true },
        },
        additionalProperties: true,
      },
    },
  },
  'x-generated-at': new Date().toISOString(),
  'x-route-count': matrix.summary.totalApiRoutes,
  'x-websockets': matrix.webSockets.map((entry) => ({
    path: entry.path,
    auth: entry.auth,
    sourceFile: entry.sourceFile,
    line: entry.line,
    notes: entry.notes,
  })),
};

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, `${JSON.stringify(spec, null, 2)}\n`);
console.log(`Generated ${outputPath} with ${Object.keys(paths).length} paths`);
