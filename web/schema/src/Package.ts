import { z } from 'zod'

export const PACKAGE_TYPES = [
  'alpine',
  'arch',
  'cargo',
  'chef',
  'composer',
  'conan',
  'conda',
  'container',
  'cran',
  'debian',
  'generic',
  'go',
  'helm',
  'maven',
  'npm',
  'nuget',
  'pub',
  'pypi',
  'rpm',
  'alt',
  'rubygems',
  'swift',
  'vagrant',
] as const

export const PackageTypeSchema = z.enum(PACKAGE_TYPES)
export type PackageType = z.infer<typeof PackageTypeSchema>

export const PackageRowSchema = z.object({
  id: z.number(),
  owner_id: z.number(),
  repo_id: z.number().nullable(),
  type: PackageTypeSchema,
  name: z.string(),
  lower_name: z.string(),
  semver_compatible: z.boolean(),
  is_internal: z.boolean(),
})
export type PackageRow = z.infer<typeof PackageRowSchema>

export const PackageVersionRowSchema = z.object({
  id: z.number(),
  package_id: z.number(),
  creator_id: z.number(),
  version: z.string(),
  lower_version: z.string(),
  metadata_json: z.string().nullable(),
  is_internal: z.boolean(),
  download_count: z.number(),
  created_unix: z.number(),
})
export type PackageVersionRow = z.infer<typeof PackageVersionRowSchema>

export const PackageBlobRowSchema = z.object({
  id: z.number(),
  hash_sha256: z.string(),
  size: z.number(),
  hash_md5: z.string().nullable(),
  hash_sha1: z.string().nullable(),
})
export type PackageBlobRow = z.infer<typeof PackageBlobRowSchema>

export const PackageFileRowSchema = z.object({
  id: z.number(),
  version_id: z.number(),
  blob_id: z.number(),
  name: z.string(),
  lower_name: z.string(),
  composite_key: z.string(),
  is_lead: z.boolean(),
  created_unix: z.number(),
})
export type PackageFileRow = z.infer<typeof PackageFileRowSchema>

export const PackagePropertyRowSchema = z.object({
  id: z.number(),
  ref_type: z.enum(['package', 'version', 'file']),
  ref_id: z.number(),
  name: z.string(),
  value: z.string(),
})
export type PackagePropertyRow = z.infer<typeof PackagePropertyRowSchema>

export const PackageInfoSchema = z.object({
  owner: z.object({ id: z.number(), name: z.string() }),
  type: PackageTypeSchema,
  name: z.string(),
  version: z.string(),
})
export type PackageInfo = z.infer<typeof PackageInfoSchema>

export const PackageTypeEntrySchema = z.object({
  type: z.string(),
  upstream: z.string(),
})
export type PackageTypeEntry = z.infer<typeof PackageTypeEntrySchema>

export const PackageTypesResponseSchema = z.object({
  types: z.array(PackageTypeEntrySchema),
})
export type PackageTypesResponse = z.infer<typeof PackageTypesResponseSchema>

export const RucoderConfigSchema = z.object({
  http_proxy: z.string().optional(),
  self_base: z.string().optional(),
})
export type RucoderConfig = z.infer<typeof RucoderConfigSchema>

export const OciCatalogSchema = z.object({
  repositories: z.array(z.string()),
})
export type OciCatalog = z.infer<typeof OciCatalogSchema>

export const UnifiedPackageEntrySchema = z.object({
  name: z.string(),
  type: z.string(),
  latest_version: z.string().nullable(),
  versions: z.number(),
})
export type UnifiedPackageEntry = z.infer<typeof UnifiedPackageEntrySchema>

export const UnifiedPackageListResponseSchema = z.object({
  ok: z.boolean(),
  data: z.object({ packages: z.array(UnifiedPackageEntrySchema) }),
})

export const PackageVersionDetailSchema = z.object({
  version: z.string(),
  download_count: z.number(),
  created_unix: z.number(),
})
export type PackageVersionDetail = z.infer<typeof PackageVersionDetailSchema>

export const PackageVersionWithFilesSchema = PackageVersionDetailSchema.extend({
  files: z.array(
    z.object({
      name: z.string(),
      size: z.number(),
      sha256: z.string(),
    }),
  ),
})
export type PackageVersionWithFiles = z.infer<
  typeof PackageVersionWithFilesSchema
>

export const PackageTypeListEntrySchema = z.object({
  name: z.string(),
  latest_version: z.string().nullable(),
  versions: z.array(PackageVersionDetailSchema),
})

export const PackageVersionsResponseSchema = z.object({
  ok: z.boolean(),
  data: z.object({
    name: z.string(),
    type: z.string(),
    versions: z.array(PackageVersionWithFilesSchema),
  }),
})
