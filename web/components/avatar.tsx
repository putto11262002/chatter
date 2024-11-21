export default function Avatar({ name }: { name: string }) {
  const firstChar = name.length > 0 ? name[0].toUpperCase() : "F";
  return (
    <div className="bg-muted border border-foreground w-9 h-9 rounded-full flex items-center justify-center">
      {firstChar}
    </div>
  );
}
