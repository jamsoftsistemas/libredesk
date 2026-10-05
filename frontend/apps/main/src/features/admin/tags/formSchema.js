import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(3, {
      message: t('admin.conversationTags.name.valid'),
    }),
  visibility: z
    .enum(['all', 'team', 'inbox'])
    .default('all'),
  team_id: z.union([z.string(), z.number()]).optional().nullable(),
  inbox_id: z.union([z.string(), z.number()]).optional().nullable(),
}).refine((data) => data.visibility !== 'team' || !!data.team_id, {
  message: t('globals.messages.required'),
  path: ['team_id'],
}).refine((data) => data.visibility !== 'inbox' || !!data.inbox_id, {
  message: t('globals.messages.required'),
  path: ['inbox_id'],
})
