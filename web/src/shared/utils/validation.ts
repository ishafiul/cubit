import { z } from 'zod';
import { Result, ok, err } from './result';

export function validateSchema<T>(
  schema: z.ZodType<T>,
  data: unknown
): Result<T, z.ZodError> {
  const parsed = schema.safeParse(data);
  if (parsed.success) {
    return ok(parsed.data);
  }
  return err(parsed.error);
}

export const environmentVariableSchema = z.object({
  key: z
    .string()
    .min(1, 'Key is required')
    .regex(/^[A-Za-z_][A-Za-z0-9_]*$/, 'Key must be a valid identifier (alphanumeric and underscores)'),
  value: z.string(),
  isSecret: z.boolean().optional().default(false),
});

export type EnvironmentVariableInput = z.infer<typeof environmentVariableSchema>;

export const resourceBindingTypeSchema = z.enum([
  'kv',
  'kv_namespace',
  'd1',
  'd1_database',
  'r2',
  'r2_bucket',
  'service',
  'queue',
  'durable_object',
  'workflow',
  'container',
  'assets',
]);

export const resourceBindingSchema = z.object({
  name: z
    .string()
    .min(1, 'Binding name is required')
    .regex(/^[A-Za-z_][A-Za-z0-9_]*$/, 'Binding name must be a valid identifier'),
  type: resourceBindingTypeSchema,
  resourceId: z.string().min(1, 'Resource ID is required'),
});

export type ResourceBindingInput = z.infer<typeof resourceBindingSchema>;

export const domainSchema = z.object({
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .regex(
      /^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/,
      'Invalid hostname format'
    ),
  pathPrefix: z.string().optional().default('/'),
});

export type DomainInput = z.infer<typeof domainSchema>;

export const createAppSchema = z
  .object({
    name: z
      .string()
      .min(1, 'Name is required')
      .regex(/^[a-z0-9-]+$/, 'Application name must be lowercase alphanumeric and hyphens'),
    sourceType: z.enum(['inline', 'git']),
    gitRepo: z.string().optional(),
    gitBranch: z.string().optional().default('main'),
    rootDir: z.string().optional(),
    inlineCode: z.string().optional(),
    autoDeploy: z.boolean().optional().default(false),
  })
  .superRefine((data, ctx) => {
    if (data.sourceType === 'git' && (!data.gitRepo || !data.gitRepo.trim())) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Git repository URL is required when source type is git',
        path: ['gitRepo'],
      });
    }
  });

export type CreateAppInput = z.infer<typeof createAppSchema>;
