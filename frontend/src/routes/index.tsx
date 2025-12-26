import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
  component: Index,
})

function Index() {
  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold">Welcome Home!</h1>
      <p className="mt-2 text-muted-foreground">
        This is the home page of your application.
      </p>
    </div>
  )
}
