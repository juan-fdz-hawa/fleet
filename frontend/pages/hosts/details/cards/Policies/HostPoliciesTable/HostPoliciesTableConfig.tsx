import React from "react";

import { IHostPolicy } from "interfaces/policy";
import { PolicyResponse, DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { noop } from "lodash";

import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

type PolicyStatus = "pass" | "fail";

interface IStatusCellValue {
  displayName: string;
  statusName: IndicatorStatus;
  value: PolicyStatus;
}
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IHostPolicy;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePolicyTableHeaders = (
  togglePolicyDetails: (policy: IHostPolicy, teamId?: number) => void,
  currentTeamId?: number
): IDataColumn[] => {
  const STATUS_CELL_VALUES: Record<PolicyStatus, IStatusCellValue> = {
    pass: {
      displayName: "Yes",
      statusName: "success",
      value: "pass",
    },
    fail: {
      displayName: "No",
      statusName: "error",
      value: "fail",
    },
  };

  return [
    {
      title: "Name",
      Header: "Name",
      accessor: "name",
      disableSortBy: true,
      Cell: (cellProps) => {
        const { name } = cellProps.row.original;

        return <LinkCell customOnClick={noop} tooltipTruncate value={name} />;
      },
    },
    {
      title: "Status",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      sortType: "caseInsensitive",
      accessor: "response",
      Cell: (cellProps) => {
        if (cellProps.row.original.response === "") {
          return <>{DEFAULT_EMPTY_CELL_VALUE}</>;
        }

        const responseValue =
          STATUS_CELL_VALUES[cellProps.row.original.response];
        return (
          <StatusIndicatorWithIcon
            value={responseValue.displayName}
            status={responseValue.statusName}
          />
        );
      },
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps) => {
        return (
          <>
            {cellProps.row.original.response && (
              <ViewAllHostsLink
                queryParams={{
                  policy_id: cellProps.row.original.id,
                  policy_response:
                    cellProps.row.original.response === "pass"
                      ? PolicyResponse.PASSING
                      : PolicyResponse.FAILING,
                  team_id: currentTeamId,
                }}
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];
};

const generatePolicyDataSet = (policies: IHostPolicy[]): IHostPolicy[] => {
  return policies;
};

export { generatePolicyTableHeaders, generatePolicyDataSet };
