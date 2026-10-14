import * as z from 'zod'

export const createFormSchema = (t) =>
  z.object({
    external_id: z.string().optional().nullable(),
    name: z
      .string({
        required_error: t('globals.messages.required')
      })
      .min(2, {
        message: t('validation.minmax', {
          min: 2,
          max: 200
        })
      })
      .max(200, {
        message: t('validation.minmax', {
          min: 2,
          max: 200
        })
      }),
    document: z.string().optional().nullable(),
    city: z.string().optional().nullable(),
    system: z.string().optional().nullable()
  })
