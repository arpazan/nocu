import { z } from "zod";

/** Mirrors identity's loginRequest; IDENTITY_MIN_PASSWORD_LENGTH defaults to 8. */
export const loginSchema = z.object({
  email: z.email("Enter a valid email address"),
  password: z.string().min(8, "Password must be at least 8 characters"),
});

export type LoginValues = z.infer<typeof loginSchema>;

/** Mirrors identity's registerRequest; name is optional and stored as given. */
export const registerSchema = loginSchema.extend({
  name: z.string().trim().max(120, "Name must be at most 120 characters").optional(),
});

export type RegisterValues = z.infer<typeof registerSchema>;
