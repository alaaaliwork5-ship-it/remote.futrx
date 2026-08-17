import type {
  ProjectContainerInfo,
  ProjectMemory,
  ProjectSecret,
} from "../../models/project";

export interface ProjectDataLoadSignal {
  cancelled: boolean;
}

export interface ProjectContainerRecord {
  loading: boolean;
  data?: ProjectContainerInfo;
  error?: string;
  refreshedAt?: number;
}

export interface SecretsRecord {
  loading: boolean;
  data?: ProjectSecret[];
  error?: string;
}

export interface AccessRecord {
  loading: boolean;
  data?: string[];
  error?: string;
}

export interface MemoryRecord {
  loading: boolean;
  data?: ProjectMemory;
  error?: string;
}
