import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import type {
  VirshSandboxInternalRestSandboxInfo,
  VirshSandboxInternalStoreCommand,
  VirshSandboxInternalStoreSandbox,
  VirshSandboxInternalAnsibleJobResponse,
} from "~/virsh-sandbox/model";

// Types for API responses
interface ListSandboxesResponse {
  sandboxes: VirshSandboxInternalRestSandboxInfo[];
  total: number;
}

interface GetSandboxResponse {
  sandbox: VirshSandboxInternalStoreSandbox;
  commands?: VirshSandboxInternalStoreCommand[];
}

interface ListSandboxCommandsResponse {
  commands: VirshSandboxInternalStoreCommand[];
  total: number;
}

// API functions
const listSandboxes = async (): Promise<ListSandboxesResponse> => {
  const response = await axios.get("/v1/sandboxes");
  return response.data;
};

const getSandbox = async (
  id: string,
  includeCommands = true
): Promise<GetSandboxResponse> => {
  const response = await axios.get(`/v1/sandboxes/${id}`, {
    params: { include_commands: includeCommands },
  });
  return response.data;
};

const listSandboxCommands = async (
  id: string
): Promise<ListSandboxCommandsResponse> => {
  const response = await axios.get(`/v1/sandboxes/${id}/commands`);
  return response.data;
};

const createAnsibleJob = async (data: {
  vm_name: string;
  playbook: string;
  check?: boolean;
}): Promise<VirshSandboxInternalAnsibleJobResponse> => {
  const response = await axios.post("/v1/ansible/jobs", data);
  return response.data;
};

// Query hooks
export const useListSandboxes = () => {
  return useQuery({
    queryKey: ["sandboxes"],
    queryFn: listSandboxes,
    select: (data) => data.sandboxes,
  });
};

export const useGetSandbox = (id: string, includeCommands = true) => {
  return useQuery({
    queryKey: ["sandbox", id, includeCommands],
    queryFn: () => getSandbox(id, includeCommands),
    enabled: !!id,
  });
};

export const useListSandboxCommands = (id: string) => {
  return useQuery({
    queryKey: ["sandbox-commands", id],
    queryFn: () => listSandboxCommands(id),
    enabled: !!id,
    refetchInterval: 5000, // Poll for new commands every 5 seconds
  });
};

// Mutation hooks
export const useCreateAnsibleJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createAnsibleJob,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ansible-jobs"] });
    },
  });
};
