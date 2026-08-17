import { useCallback, useState } from "preact/hooks";
import { projectMemoryApi } from "../../../api/project/projectMemoryApi";
import type { ProjectMemory, ProjectMeta } from "../../../models/project";
import type {
  ProjectDataLoadSignal,
  MemoryRecord,
} from "../../projects/projectContainerRecords";

export function useProjectMemory(project: ProjectMeta | null) {
  const [record, setRecord] = useState<MemoryRecord>({ loading: false });

  const load = useCallback(
    async (signal?: ProjectDataLoadSignal) => {
      if (!project) {
        setRecord({ loading: false });
        return;
      }
      setRecord((current) => ({ ...current, loading: true, error: undefined }));
      try {
        const data = await projectMemoryApi.getMemory(project.id);
        if (signal?.cancelled) return;
        setRecord({ loading: false, data });
      } catch (error) {
        if (signal?.cancelled) return;
        setRecord({ loading: false, error: (error as Error).message });
      }
    },
    [project]
  );

  const save = useCallback(
    async (content: string, enabled: boolean): Promise<ProjectMemory> => {
      if (!project) throw new Error("No project selected");
      const saved = await projectMemoryApi.setMemory(
        project.id,
        content,
        enabled
      );
      setRecord({ loading: false, data: saved });
      return saved;
    },
    [project]
  );

  return { record, load, save };
}
