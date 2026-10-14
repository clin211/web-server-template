import { z } from "zod";

export const requestIdSchema = z.string().regex(/^[A-Za-z0-9._:-]{1,128}$/);

export type RequestId = z.infer<typeof requestIdSchema>;
