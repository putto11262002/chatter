import { cva, VariantProps } from "class-variance-authority";

const alertVariants = cva("px-3 py-2 rounded-md text-sm", {
  variants: {
    variant: {
      error: "text-red-800 bg-red-200 border",
      success: "text-green-800 bg-green-200 border",
      warning: "text-yellow-800 bg-yellow-200 border",
    },
  },
  defaultVariants: {
    variant: "error",
  },
});

type AlertVariantsProps = VariantProps<typeof alertVariants>;

export default function Alert({
  message,
  variant,
}: {
  message: string;
  variant?: AlertVariantsProps["variant"];
}) {
  return <div className={alertVariants({ variant })}>{message}</div>;
}
