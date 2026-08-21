import type { ModuleInfo } from './modules'

interface Props {
  module: ModuleInfo
  onClick: () => void
}

export function ModuleCard({ module, onClick }: Props) {
  return (
    <div
      onClick={onClick}
      className="group cursor-pointer rounded-xl border border-neutral-800 bg-neutral-900 p-5 transition hover:border-neutral-600 hover:bg-neutral-800"
    >
      <div className="mb-2 flex items-center justify-between">
        <h4 className="font-semibold text-white group-hover:text-blue-400">
          {module.name}
        </h4>
        {module.providers && (
          <span className="text-xs text-neutral-500">
            {module.providers.length} providers
          </span>
        )}
      </div>
      <p className="mb-3 text-sm text-neutral-400 line-clamp-2">
        {module.description}
      </p>
      <code className="text-xs text-neutral-500 break-all">
        {module.path}
      </code>
      {module.providers && (
        <div className="mt-3 flex flex-wrap gap-1">
          {module.providers.slice(0, 5).map((p) => (
            <span
              key={p}
              className="rounded bg-neutral-800 px-1.5 py-0.5 text-[10px] text-neutral-400"
            >
              {p}
            </span>
          ))}
          {module.providers.length > 5 && (
            <span className="text-[10px] text-neutral-600">
              +{module.providers.length - 5}
            </span>
          )}
        </div>
      )}
    </div>
  )
}
