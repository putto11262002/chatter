import { z } from "zod";

export class ApiError extends Error {
  code: number;
  constructor(code: number, message: string) {
    super(message);
    this.code = code;
  }
}

export const apiErorSchema = z.object({
  code: z.number(),
  message: z.string(),
});

// parse an unkown object into an ApiError
// If the object is not an ApiError, return ApiError(-1, "Unknown error")
export function parseApiError(error: unknown): ApiError {
  console.log(error);
  const parsed = apiErorSchema.safeParse(error);
  if (parsed.success) {
    return new ApiError(parsed.data.code, parsed.data.message);
  }
  return new ApiError(-1, "Unknown error");
}
