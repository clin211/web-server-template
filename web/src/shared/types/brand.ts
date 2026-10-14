export type Brand<T, TBrand extends string> = T & { readonly __brand: TBrand };

export type BusinessId = Brand<string, "BusinessId">;
