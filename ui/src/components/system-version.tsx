import { useEffect, useState } from "react"

type VersionInfo = {
    version: string
    commit: string
    build: string
}


export function SystemVersion() {
    const [info, setInfo] = useState<VersionInfo | null>(null)
  
    useEffect(() => {
      fetch("/api/version")
        .then((res) => {
          if (!res.ok) throw new Error("Failed to fetch version")
          return res.json()
        })
        .then((data) => setInfo(data))
        .catch(() => {
          setInfo({ version: "unknown", commit: "-", build: "-" })
        })
    }, [])
  
    if (!info) return <p className="text-sm text-muted-foreground">Loading version...</p>
  
    return (
      <p className="text-sm text-muted-foreground">
        Pentora v{info.version} ({info.commit} @ {info.build})
      </p>
    )
}