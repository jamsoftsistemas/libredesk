import * as z from 'zod'

export const createFormSchema = (t) =>
  z.object({
    name: z.string().min(1, t('globals.messages.required')),
    enabled: z.boolean().optional(),
    csat_enabled: z.boolean().optional(),
    prompt_tags_on_reply: z.boolean().optional(),
    reopen_window_hours: z.coerce.number().int().min(0).optional(),
    config: z
      .object({
        base_url: z.string().min(1, t('globals.messages.required')),
        instance_name: z.string().optional(),
        instance_token: z.string().optional(),
        admin_token: z.string().optional()
      })
      .refine((config) => config.instance_token || (config.admin_token && config.instance_name), {
        message: t('admin.inbox.uazapi.error.missingCreateFields'),
        path: ['instance_token']
      })
  })
